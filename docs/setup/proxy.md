# Proxy

This document provides high-level instructions for configuring Drone to work with a corporate proxy server.

## Http Proxy

The HTTP_PROXY environment variable holds the hostname or IP address of your proxy server. You can specify the HTTP_PROXY variables in your `/etc/drone/dronerc` file or as an envronment variable.

```
HTTPS_PROXY=https://proxy.example.com
HTTP_PROXY=http://proxy.example.com
```

These variables are propogated throughout your build environment, including build and plugin containers. To verify the environment variables are being set in your build container you can add the `env` command to your build script.

We also recommend you provide both uppercase and lowercase environment variables. We've found that certain common unix tools are case-sensitive:

```
HTTP_PROXY=http://proxy.example.com
http_proxy=http://proxy.example.com
```

## No Proxy

The `NO_PROXY` variable should contain a comma-separated list of domain extensions the proxy should not be used for. This typically includes resources inside your network, such as your GitHub Enterprise server.

```
NO_PROXY=.example.com, *.docker.example.com
```

You may also need to add your Docker daemon hostnames to the above list.