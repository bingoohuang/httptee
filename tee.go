package httptee

import (
	"crypto/tls"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CloneURL clones a URL.
func CloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}

	u2 := new(url.URL)
	*u2 = *u

	if u.User != nil {
		u2.User = new(url.Userinfo)
		*u2.User = *u.User
	}

	return u2
}

// SetRequestTarget sets the req URL.
// this turns a inbound req (a req without URL) into an outbound req.
func SetRequestTarget(request *http.Request, b Backend) {
	request.URL.Scheme = b.Scheme
	request.URL.Host = b.Host
}

// Setup setup handler.
func (h *Handler) Setup(primaryTarget string, alternateWorkers, alternateChanSize int) {
	h.randomizer = *rand.New(rand.NewSource(time.Now().UnixNano()))
	h.Primary = schemeAndHost(primaryTarget)
	h.primaryTransport = MakeTransport(h.PrimaryTimeout, h.CloseConnections)
	h.alterTransport = MakeTransport(h.AlternateTimeout, h.CloseConnections)

	if len(h.Alternatives) > 0 {
		h.alterRequestChan = make(chan AlternativeReq, alternateChanSize)

		StartWorkers(alternateWorkers, func() {
			for req := range h.alterRequestChan {
				h.handleAlterRequest(req)
			}
		})
	}
}

// MakeTransport makes a new http.Transport.
func MakeTransport(t time.Duration, closeConnections bool) *http.Transport {
	return &http.Transport{
		DialContext:           (&net.Dialer{Timeout: t, KeepAlive: 10 * t}).DialContext,
		DisableKeepAlives:     closeConnections,
		TLSHandshakeTimeout:   t,
		ResponseHeaderTimeout: t,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, // nolint
	}
}

// handleAlterRequest duplicate req and sent it to alternative Backend
func (h *Handler) handleAlterRequest(r AlternativeReq) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in ServeHTTP(alternate req) from:", r)
		}
	}()

	if rsp := handleRequest(r.req, h.alterTransport); rsp != nil {
		_ = rsp.Body.Close()
	}
}

// handleRequest sends a req and returns the response.
func handleRequest(request *http.Request, t http.RoundTripper) (rsp *http.Response) {
	var err error

	if rsp, err = t.RoundTrip(request); err != nil {
		log.Println("Request failed:", err)
	}

	return
}

// schemeAndHost parse URL into scheme and rest of endpoint
func schemeAndHost(url string) Backend {
	if strings.HasPrefix(url, "https") {
		return Backend{Scheme: "https", Host: strings.TrimPrefix(url, "https://")}
	}

	return Backend{Scheme: "http", Host: strings.TrimPrefix(url, "http://")}
}
