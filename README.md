# Dynamic Upstreams Caddy Module

> For AWS Autoscaling Groups.

The main purpose of this project is to add a module under namespace `http.reverse_proxy.upstreams`.

The overall scope of this project is similar to [`nginx-asg-async`](https://github.com/nginxinc/nginx-asg-sync) which
is similar counter part for Nginx Plus.

## Configuration

### JSON Configuration:

```json
{
    "source": "asg",
    "provider": "aws",
    "precache": true,
    "port": 8000,
    "cache_interval_seconds": 30,
    "aws_config": {
        "region": "us-east-1",
        "profile": "production",
        "asg_name": "myasg",
        "with_in_service": true
    }
}
```

### Fields

* `source`: mandatory to specify which module this is for.
* `provider`: specifies what provider to use, like AWS for now.
* `precache`: when set true will update cache at the time of provisioning.
* `port`: specifies the port to connect to or use in `Dial`
* `cache_interval_seconds`: specifies how much time it should wait before rerunning the GetUpstreams call for provider.
* `aws_config`: is needed when using `aws` as provider.
    * `region`: overrides AWS region than default.
    * `profile`: specifies the profile to use in case of shared credentials.
    * `asg_name`: is used to filter the instances by tag value.
    * `with_with_in_service`: when set to true will filter instances only with lifecycle state as InService.
