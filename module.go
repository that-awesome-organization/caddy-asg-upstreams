package asgupstreams

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"development.thatwebsite.xyz/caddy/asgupstreams/awsclient"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"go.uber.org/zap"
)

var (
	cache = &cachedUpstreams{
		cachedTill: time.Now().Add(-time.Hour),
		values:     []*reverseproxy.Upstream{},
	}
)

func init() {
	caddy.RegisterModule(AutoScalingGroupUpstreams{})
}

// AutoScalingGroupUpstreams provides upstreams from AWS's Application
// Load Balancer target group's registered targets.
type AutoScalingGroupUpstreams struct {
	// Provider specifies what provider to use, like AWS for now
	Provider string `json:"provider,omitempty"`

	// Port specifies the port to connect to or use in Dial()
	Port int `json:"port,omitempty"`

	// CacheIntervalSeconds specifies how much time it should wait
	// before rerunning the GetUpstreams call for provider.
	CacheIntervalSeconds int `json:"cache_interval_seconds,omitempty"`

	// AWSConfig specifies the details on AWS connection, like region,
	// profile and autoscaling group name.
	AWSConfig *awsclient.AWSConfig `json:"aws_config,omitempty"`

	awsc *awsclient.AWSClient

	logger *zap.Logger
}

type cachedUpstreams struct {
	cachedTill time.Time
	values     []*reverseproxy.Upstream
	mu         sync.Mutex
}

// CaddyModule returns the Caddy module information.
func (AutoScalingGroupUpstreams) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.reverse_proxy.upstreams.asg",
		New: func() caddy.Module { return new(AutoScalingGroupUpstreams) },
	}
}

func (au *AutoScalingGroupUpstreams) Provision(ctx caddy.Context) error {
	au.logger = ctx.Logger(au)
	if au.Provider != "aws" {
		return fmt.Errorf("invalid provider: %q", au.Provider)
	}

	if au.CacheIntervalSeconds == 0 {
		au.CacheIntervalSeconds = 5
	}

	switch au.Provider {
	case "aws":
		if err := au.AWSConfig.Validate(); err != nil {
			return err
		}
		if awsc, err := awsclient.New(ctx, au.AWSConfig, au.logger); err != nil {
			return err
		} else {
			au.awsc = awsc
		}

	}
	return nil
}

func (au *AutoScalingGroupUpstreams) GetUpstreams(r *http.Request) ([]*reverseproxy.Upstream, error) {
	// if cache is still valid use the same
	if cache.cachedTill.After(time.Now()) {
		return cache.values, nil
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()
	if au.awsc != nil {
		upstreams, err := au.awsc.GetUpstreams(r.Context(), au.Port)
		if err != nil {
			au.logger.Error("error in GetUpstreams", zap.Error(err))
			return nil, err
		}
		cache.cachedTill = time.Now().Add(time.Second * time.Duration(au.CacheIntervalSeconds))
		cache.values = upstreams
	}
	return cache.values, nil
}

var (
	_ caddy.Provisioner           = (*AutoScalingGroupUpstreams)(nil)
	_ reverseproxy.UpstreamSource = (*AutoScalingGroupUpstreams)(nil)
)
