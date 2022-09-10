# Caddy Module for Auto Scaling Groups as Dynamic Upstreams

The main purpose of this project is to add a module under namespace `http.reverse_proxy.upstreams`.

The overall scope of this project is similar to [`nginx-asg-async`](https://github.com/nginxinc/nginx-asg-sync) that is
similar counter part for Nginx Plus.


## Configuration

JSON Configuration:

```json
{
    "provider": "aws",
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