---
title: Using Pipeleek with Proxies
description: Learn how to configure Pipeleek to work with HTTP and SOCKS5 proxies for intercepting traffic, testing, and accessing internal networks.
keywords:
  - proxy
  - HTTP proxy
  - SOCKS5 proxy
  - Burp Suite
  - traffic interception
  - proxy configuration
  - HTTP_PROXY
  - network proxy
---

Pipeleek supports routing all HTTP/HTTPS traffic through a proxy server. This is useful for:

- **Traffic interception**: Analyze API calls with tools like Burp Suite or mitmproxy
- **Internal network access**: Connect to internal GitLab/Gitea/... instances through SOCKS5 proxies

## Proxy Configuration

Pipeleek uses the standard `HTTP_PROXY` environment variable for proxy configuration.

```bash
HTTP_PROXY=http://127.0.0.1:8080 pipeleek gl scan -g https://gitlab.com -t glpat-xxxxx
```

SOCKS5 can be used as well.

```bash
HTTP_PROXY=socks5://127.0.0.1:1080 pipeleek gl scan -g https://gitlab.internal.company.com -t glpat-xxxxx
```

## TLS/SSL

Pipeleek automatically skips TLS certificate verification (required for self signed certificates).
