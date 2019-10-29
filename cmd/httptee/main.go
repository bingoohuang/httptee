package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/bingoohuang/httptee"
)

func main() {
	var altServers httptee.Backends

	flag.Var(&altServers, "b", "where testing traffic goes. response are skipped. "+
		"http://localhost:8081/test, allowed multiple times for multiple testing backends")

	listen := flag.String("l", ":8888", "port to accept requests")
	primaryTarget := flag.String("a", "", "primary endpoint. eg. http://localhost:8080/production")
	primaryTimeout := flag.Int("a.timeout", 2500, "timeout in millis for primary traffic")
	alterTimeout := flag.Int("b.timeout", 1000, "timeout in millis for alternate traffic")
	alternateWorkers := flag.Int("b.workers", 0, "alternate workers, default to cpu cores")
	alternateChanSize := flag.Int("b.chanSize", 1000, "alternate workers chan size")
	primaryHostRewrite := flag.Bool("a.rewrite", false, "rewrite host header for primary traffic")
	alterHostRewrite := flag.Bool("b.rewrite", false, "rewrite host header for alternate traffic")
	percent := flag.Int("b.percent", 100, "percentage of traffic to alternate")
	tlsPrivateKey := flag.String("key.file", "", "TLS private key file path")
	tlsCertificate := flag.String("cert.file", "", "TLS certificate file path")
	forwardClientIP := flag.Bool("forward-client-ip", false,
		"forwarding of the client IP to the backend using 'X-Forwarded-For' and 'Forwarded' headers")
	closeConns := flag.Bool("close-connections", false, "close connections to clients and backends")

	flag.Parse()

	log.Printf("Starting httptee at %s sending to A: %s and B: %s",
		*listen, *primaryTarget, altServers)

	h := &httptee.Handler{
		Alternatives:         altServers,
		ForwardClientIP:      *forwardClientIP,
		Percent:              *percent,
		AlternateTimeout:     time.Duration(*alterTimeout) * time.Millisecond,
		AlternateHostRewrite: *alterHostRewrite,
		PrimaryHostRewrite:   *primaryHostRewrite,
		PrimaryTimeout:       time.Duration(*primaryTimeout) * time.Millisecond,
		CloseConnections:     *closeConns,
	}

	h.Setup(*primaryTarget, *alternateWorkers, *alternateChanSize)

	server := &http.Server{Handler: h}

	if *closeConns { // Close connections to clients by setting the "Connection": "close" header in the response.
		server.SetKeepAlivesEnabled(false)
	}

	listener := createListener(*tlsPrivateKey, *tlsCertificate, *listen)
	if err := server.Serve(listener); err != nil {
		panic(err)
	}
}

func createListener(tlsPrivateKey, tlsCertificate, listen string) (listener net.Listener) {
	var err error

	if tlsPrivateKey != "" {
		cer, err := tls.LoadX509KeyPair(tlsCertificate, tlsPrivateKey)
		if err != nil {
			log.Fatalf("Failed to load certficate: %s and private key: %s", tlsCertificate, tlsPrivateKey)
		}

		config := &tls.Config{Certificates: []tls.Certificate{cer}}
		if listener, err = tls.Listen("tcp", listen, config); err != nil {
			log.Fatalf("Failed to listen to %s: %s", listen, err)
		}
	} else if listener, err = net.Listen("tcp", listen); err != nil {
		log.Fatalf("Failed to listen to %s: %s", listen, err)
	}

	return
}
