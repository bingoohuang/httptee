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


1. `go get github.com/bingoohuang/golang-trial/cmd/gohttpd`
1. `go get github.com/bingoohuang/httptee/cmd/httptee`
1. `go get github.com/rakyll/hey`

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

```bash
$ httptee -h
Usage of httptee:
  -a string
    	primary endpoint. eg. http://localhost:8080/production
  -a.rewrite
    	rewrite host header for primary traffic
  -a.timeout int
    	timeout in millis for primary traffic (default 2500)
  -b value
    	where testing traffic goes. response are skipped. http://localhost:8081/test, allowed multiple times for multiple testing backends
  -b.chanSize int
    	alternate workers chan size (default 1000)
  -b.percent int
    	percentage of traffic to alternate (default 100)
  -b.rewrite
    	rewrite host header for alternate traffic traffic
  -b.timeout int
    	timeout in millis for alternate traffic (default 1000)
  -b.workers int
    	alternate workers, default to cpu cores
  -cert.file string
    	TLS certificate file path
  -close-connections
    	close connections to clients and backends
  -forward-client-ip
    	forwarding of the client IP to the backend using 'X-Forwarded-For' and 'Forwarded' headers
  -key.file string
    	TLS private key file path
  -l string
    	port to accept requests (default ":8888")
```


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

Influxdb dual writing benchmarking
-------------

1. 建立两个节点 `docker run --name influxdb1 -p 18086:8086 -d  influxdb:1.5.4`， `docker run --name influxdb2 -p 18087:8086 -d  influxdb:1.5.4`
1. 在两个节点上建立test database，下面以influxdb1为例，需要在influxdb2同理执行。

    ```bash
    ➜ docker exec -it influxdb1 bash
    root@4c638b3e019c:/# influx
    Connected to http://localhost:8086 version 1.5.4
    InfluxDB shell version: 1.5.4
    > create database test;
    > show databases;
    name: databases
    name
    ----
    _internal
    test
    ```

1. 使用httpie验证

    ```bash
    ➜ echo "cpu,host=server02,region=uswest value=1 1434055561000000000" | http http://localhost:18086/write?db=test
    HTTP/1.1 204 No Content
    Content-Type: application/json
    Date: Wed, 30 Oct 2019 02:49:29 GMT
    Request-Id: e47d9b96-fabf-11e9-8011-000000000000
    X-Influxdb-Build: OSS
    X-Influxdb-Version: 1.5.4
    X-Request-Id: e47d9b96-fabf-11e9-8011-000000000000
    
    
    
    ```

1. 构建httptee双写代理 `httptee -l :19000 -a http://localhost:18086 -b http://localhost:18087`
1. 构建influxdb-relay代理 `go get -u github.com/vente-privee/influxdb-relay`, `influxdb-relay -config=relay.toml`
    
    ```toml
    # relay.toml
    # InfluxDB && Prometheus
    [[http]]
    name = "example-http-influxdb"
    bind-addr = "0.0.0.0:9096"
    
    [[http.output]]
    name = "local-influxdb01"
    location = "http://127.0.0.1:18086/"
    endpoints = {write="/write", write_prom="/api/v1/prom/write", ping="/ping", query="/query"}
    timeout = "10s"
    
    [[http.output]]
    name = "local-influxdb02"
    location = "http://127.0.0.1:18087/"
    endpoints = {write="/write", write_prom="/api/v1/prom/write", ping="/ping", query="/query"}
    timeout = "10s"
    ```
        
1. hey单次验证 `hey -n 1 -c 1  -m POST -d  "cpu,host=server02,region=uswest value=1 1434055561000000000" "http://localhost:19000/write?db=test"`
1. hey直连压测 `hey -z 10s -q 3000 -n 100000 -c 1 -t 1 -m POST -d  "cpu,host=server02,region=uswest value=1 1434055561000000000" "http://localhost:19000/write?db=test"`
1. siege脚本 `siege -c20 -r1000 "http://127.0.0.1:9096/write?db=test POST cpu,host=server02,region=uswest value=1 1434055561000000000"`


```bash
➜  hey -z 10s -q 600 -n 100000 -c 1 -t 1 -m POST -d  "cpu,host=server02,region=uswest value=1 1434055561000000000" "http://localhost:18086/write?db=test"

Summary:
  Total:	10.0018 secs
  Slowest:	0.0071 secs
  Fastest:	0.0015 secs
  Average:	0.0019 secs
  Requests/sec:	517.5057


Response time histogram:
  0.002 [1]	|
  0.002 [4609]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.003 [523]	|■■■■■
  0.003 [28]	|
  0.004 [5]	|
  0.004 [2]	|
  0.005 [4]	|
  0.005 [0]	|
  0.006 [2]	|
  0.007 [1]	|
  0.007 [1]	|


Latency distribution:
  10% in 0.0017 secs
  25% in 0.0018 secs
  50% in 0.0019 secs
  75% in 0.0020 secs
  90% in 0.0021 secs
  95% in 0.0022 secs
  99% in 0.0026 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.0015 secs, 0.0071 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0010 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0001 secs
  resp wait:	0.0019 secs, 0.0015 secs, 0.0071 secs
  resp read:	0.0000 secs, 0.0000 secs, 0.0003 secs

Status code distribution:
  [204]	5176 responses



➜  hey -z 10s -q 600 -n 100000 -c 1 -t 1 -m POST -d  "cpu,host=server02,region=uswest value=1 1434055561000000000" "http://localhost:19000/write?db=test"

Summary:
  Total:	10.0018 secs
  Slowest:	0.0416 secs
  Fastest:	0.0019 secs
  Average:	0.0028 secs
  Requests/sec:	358.9361


Response time histogram:
  0.002 [1]	|
  0.006 [3578]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.010 [8]	|
  0.014 [2]	|
  0.018 [0]	|
  0.022 [0]	|
  0.026 [0]	|
  0.030 [0]	|
  0.034 [0]	|
  0.038 [0]	|
  0.042 [1]	|


Latency distribution:
  10% in 0.0023 secs
  25% in 0.0024 secs
  50% in 0.0026 secs
  75% in 0.0030 secs
  90% in 0.0034 secs
  95% in 0.0037 secs
  99% in 0.0047 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.0019 secs, 0.0416 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0010 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0002 secs
  resp wait:	0.0027 secs, 0.0019 secs, 0.0414 secs
  resp read:	0.0000 secs, 0.0000 secs, 0.0001 secs

Status code distribution:
  [204]	3590 responses

➜  hey -z 10s -q 600 -n 100000 -c 1 -t 1 -m POST -d  "cpu,host=server02,region=uswest value=1 1434055561000000000" "http://localhost:9096/write?db=test"

Summary:
  Total:	10.0050 secs
  Slowest:	0.0800 secs
  Fastest:	0.0017 secs
  Average:	0.0023 secs
  Requests/sec:	433.2848


Response time histogram:
  0.002 [1]	|
  0.010 [4332]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.017 [0]	|
  0.025 [1]	|
  0.033 [0]	|
  0.041 [0]	|
  0.049 [0]	|
  0.057 [0]	|
  0.064 [0]	|
  0.072 [0]	|
  0.080 [1]	|


Latency distribution:
  10% in 0.0020 secs
  25% in 0.0021 secs
  50% in 0.0022 secs
  75% in 0.0024 secs
  90% in 0.0026 secs
  95% in 0.0028 secs
  99% in 0.0032 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.0017 secs, 0.0800 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0012 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0004 secs
  resp wait:	0.0022 secs, 0.0016 secs, 0.0800 secs
  resp read:	0.0000 secs, 0.0000 secs, 0.0005 secs

Status code distribution:
  [204]	4335 responses

```
