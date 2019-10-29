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

// SetRequestTarget sets the req URL.
// this turns a inbound req (a req without URL) into an outbound req.
func SetRequestTarget(request *http.Request, b Backend) {
	if URL, err := url.Parse(b.Scheme + "://" + b.Endpoint + request.URL.String()); err != nil {
		log.Println(err)
	} else {
		request.URL = URL
	}
}

// Setup setup handler.
func (h *Handler) Setup(primaryTarget string, alternateWorkers, alternateChanSize int) {
	h.randomizer = *rand.New(rand.NewSource(time.Now().UnixNano()))
	h.Primary = SchemeAndHost(primaryTarget)
	h.primaryTransport = MakeTransport(h.PrimaryTimeout, h.CloseConnections)
	h.alterTransport = MakeTransport(h.AlternateTimeout, h.CloseConnections)

	if len(h.Alternatives) > 0 {
		h.alterRequestChan = make(chan AlternativeReq, alternateChanSize)

		StartWorkers(alternateWorkers, func() {
			for req := range h.alterRequestChan {
				h.HandleAlterRequest(req)
			}
		})
	}
}

// MakeTransport makes a new http.Transport.
func MakeTransport(timeout time.Duration, closeConnections bool) *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{ // go1.8 deprecated: Use DialContext instead
			Timeout:   timeout,
			KeepAlive: 10 * timeout,
		}).DialContext,
		DisableKeepAlives:     closeConnections,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, // nolint
	}
}

// HandleAlterRequest duplicate req and sent it to alternative Backend
func (h *Handler) HandleAlterRequest(r AlternativeReq) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in ServeHTTP(alternate req) from:", r)
		}
	}()

	if rsp := HandleRequest(r.req, h.alterTransport); rsp != nil {
		_ = rsp.Body.Close()
	}
}

// HandleRequest sends a req and returns the response.
func HandleRequest(request *http.Request, t http.RoundTripper) (rsp *http.Response) {
	var err error

	if rsp, err = t.RoundTrip(request); err != nil {
		log.Println("Request failed:", err)
	}

	return
}

// SchemeAndHost parse URL into scheme and rest of endpoint
func SchemeAndHost(url string) Backend {
	if strings.HasPrefix(url, "https") {
		return Backend{Scheme: "https", Endpoint: strings.TrimPrefix(url, "https://")}
	}

	return Backend{Scheme: "http", Endpoint: strings.TrimPrefix(url, "http://")}
}
