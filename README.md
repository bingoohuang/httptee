httptee
=========

A reverse HTTP proxy that duplicates requests, original from [teeproxy](https://github.com/chrislusf/teeproxy).

Why you may need this?
----------------------

You may have production servers running, but you need to upgrade to a new system. You want to run A/B test on both old and new systems to confirm the new system can handle the production load, and want to see whether the new system can run in shadow mode continuously without any issue.

How it works?
-------------

httptee is a reverse HTTP proxy. For each incoming request, it clones the request into 2 requests, forwards them to 2 servers. The results from server A are returned as usual, but the results from server B are ignored.

httptee handles GET, POST, and all other http methods.

Build
-------------

1. local `go install ./...`
1. linux `env GOOS=linux GOARCH=amd64 go install ./...`

Demo
-------------

Using [hey](https://github.com/rakyll/hey) to do as HTTP load generator. 

> I’m using hey tool because it’s simple and allows generating constant load instead of bombarding as hard as possible like many other tools do (wrk, apache benchmark, siege).
> [nginx mirroring tips and tricks](https://alex.dzyoba.com/blog/nginx-mirror/)

> 我用 hey 来测试压力，因为它很简单，可以施加稳定的负载，其他工具的负载施加很不稳定（例如，wrk, apache benchmark, siege）。
> [Nginx 流量镜像使用技巧](https://mp.weixin.qq.com/s/GpFafOWtlKIrmEftKE-Vdg)

```bash
➜ gohttpd -p 9001 &
[1] 2033
➜ gohttpd -p 9002 &
[2] 2110
➜ httptee -l :9900 -a http://localhost:9001 -b http://localhost:9002  &
[1] 5496
2019/10/29 17:59:40 Starting httptee at :9900 sending to A: http://localhost:9001 and B: [{localhost:9002 http}]
➜ hey -z 10s -q 1700 -n 100000 -c 1 -t 1 http://127.0.0.1:9900/say
  
  Summary:
    Total:	10.0020 secs
    Slowest:	0.0408 secs
    Fastest:	0.0002 secs
    Average:	0.0002 secs
    Requests/sec:	1661.5623
  
    Total data:	731236 bytes
    Size/request:	44 bytes
  
  Response time histogram:
    0.000 [1]	|
    0.004 [16610]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
    0.008 [0]	|
    0.012 [0]	|
    0.016 [0]	|
    0.020 [0]	|
    0.025 [3]	|
    0.029 [2]	|
    0.033 [2]	|
    0.037 [0]	|
    0.041 [1]	|
  
  
  Latency distribution:
    10% in 0.0002 secs
    25% in 0.0002 secs
    50% in 0.0002 secs
    75% in 0.0002 secs
    90% in 0.0002 secs
    95% in 0.0003 secs
    99% in 0.0005 secs
  
  Details (average, fastest, slowest):
    DNS+dialup:	0.0000 secs, 0.0002 secs, 0.0408 secs
    DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0000 secs
    req write:	0.0000 secs, 0.0000 secs, 0.0001 secs
    resp wait:	0.0002 secs, 0.0001 secs, 0.0407 secs
    resp read:	0.0000 secs, 0.0000 secs, 0.0003 secs
  
  Status code distribution:
    [200]	16619 responses
```

Usage
-------------

```
 ./httptee -l :8888 -a [http(s)://]localhost:9000 -b [http(s)://]localhost:9001 [-b [http(s)://]localhost:9002]
```

`-l` specifies the listening port. `-a` and `-b` are meant for system A and systems B. The B systems can be taken down or started up without causing any issue to the httptee.

#### Configuring timeouts ####
 
It's also possible to configure the timeout to both systems

*  `-a.timeout int`: timeout in milliseconds for production traffic (default `2500`)
*  `-b.timeout int`: timeout in milliseconds for alternate site traffic (default `1000`)

#### Configuring host header rewrite ####

Optionally rewrite host value in the http request header.

*  `-a.rewrite bool`: rewrite for production traffic (default `false`)
*  `-b.rewrite bool`: rewrite for alternate site traffic (default `false`)
 
#### Configuring a percentage of requests to alternate site ####

*  `-p float64`: only send a percentage of requests. The value is float64 for more precise control. (default `100.0`)

#### Configuring HTTPS ####

*  `-key.file string`: a TLS private key file. (default `""`)
*  `-cert.file string`: a TLS certificate file. (default `""`)

#### Configuring client IP forwarding ####

It's possible to write `X-Forwarded-For` and `Forwarded` header (RFC 7239) so
that the production and alternate backends know about the clients:

*  `-forward-client-ip` (default is false)

#### Configuring connection handling ####

By default, httptee tries to reuse connections. This can be turned off, if the
endpoints do not support this.

*  `-close-connections` (default is false)

