package awsclient

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"go.uber.org/zap"
)

type AWSConfig struct {
	Region  string `json:"region,omitempty"`
	Profile string `json:"profile,omitempty"`

	AutoScalingGroupName string `json:"asg_name,omitempty"`
	WithInService        bool   `json:"with_in_service,omitempty"`
}

func (ac *AWSConfig) Validate() error {
	if ac.AutoScalingGroupName == "" {
		return fmt.Errorf("empty autoscaling group name")
	}
	return nil
}

type AWSClient struct {
	clientASG *autoscaling.Client
	clientEC2 *ec2.Client

	asgName       string
	withInService bool
	logger        *zap.Logger
}

func New(ctx context.Context, awsconfig *AWSConfig, logger *zap.Logger) (*AWSClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(awsconfig.Profile),
		config.WithRegion(awsconfig.Region),
	)

	if err != nil {
		return nil, err
	}

	return &AWSClient{
		clientASG:     autoscaling.NewFromConfig(cfg),
		clientEC2:     ec2.NewFromConfig(cfg),
		asgName:       awsconfig.AutoScalingGroupName,
		withInService: awsconfig.WithInService,
		logger:        logger,
	}, nil
}

func (awsclient *AWSClient) GetUpstreams(ctx context.Context, port int) ([]*reverseproxy.Upstream, error) {
	result, err := awsclient.clientEC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:aws:autoscaling:groupName"),
				Values: []string{awsclient.asgName},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(result.Reservations) == 0 {
		return nil, fmt.Errorf("no instances found for autoscaling group")
	}

	insMap := map[string]string{}
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.InstanceId != nil && instance.PrivateIpAddress != nil {
				insMap[*instance.InstanceId] = *instance.PrivateIpAddress
			}
		}
	}

	upstreams := []*reverseproxy.Upstream{}

	if awsclient.withInService {
		for _, privateIpAddress := range awsclient.getInServiceInstances(ctx, insMap) {
			upstreams = append(upstreams, &reverseproxy.Upstream{
				Dial: fmt.Sprintf("%s:%d", privateIpAddress, port),
			})
		}
	} else {
		for _, privateIpAddress := range insMap {
			upstreams = append(upstreams, &reverseproxy.Upstream{
				Dial: fmt.Sprintf("%s:%d", privateIpAddress, port),
			})
		}
	}

	return upstreams, nil
}

func (awsclient *AWSClient) getInServiceInstances(ctx context.Context, insMap map[string]string) []string {
	maxRecords := 10
	batches := [][]string{}

	i := 0
	for k := range insMap {
		if len(batches[i]) == maxRecords {
			i++
		}
		batches[i] = append(batches[i], k)
	}
	inServiceIPs := []string{}
	for _, batch := range batches {
		output, err := awsclient.clientASG.DescribeAutoScalingInstances(ctx, &autoscaling.DescribeAutoScalingInstancesInput{
			InstanceIds: batch,
		})
		if err != nil {
			return nil
		}
		for _, i := range output.AutoScalingInstances {
			if *i.LifecycleState == "InService" {
				inServiceIPs = append(inServiceIPs, insMap[*i.InstanceId])
			}
		}
	}

	return inServiceIPs
}
