package httptee

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Sets the req URL.
//
// This turns a inbound req (a req without URL) into an outbound req.
func setRequestTarget(request *http.Request, target string, scheme string) {
	URL, err := url.Parse(scheme + "://" + target + request.URL.String())
	if err != nil {
		log.Println(err)
	}

	request.URL = URL
}

func (h *Handler) getTransport(scheme string, timeout time.Duration) *http.Transport {
	h.transportCacheLock.RLock()
	if t, ok := h.TransportCache[TransportCacheKey{Scheme: scheme, Timeout: timeout}]; ok {
		h.transportCacheLock.RUnlock()
		return t
	}

	h.transportCacheLock.RUnlock()
	h.transportCacheLock.Lock()

	var t *http.Transport

	if scheme == "https" {
		t = &http.Transport{
			Dial: (&net.Dialer{ // go1.8 deprecated: Use DialContext instead
				Timeout:   timeout,
				KeepAlive: 10 * timeout,
			}).Dial,
			DisableKeepAlives:     h.CloseConnections,
			TLSHandshakeTimeout:   timeout,
			ResponseHeaderTimeout: timeout,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, // nolint
		}
	} else {
		t = &http.Transport{
			Dial: (&net.Dialer{ // go1.8 deprecated: Use DialContext instead
				Timeout:   timeout,
				KeepAlive: 10 * timeout,
			}).Dial,
			DisableKeepAlives:     h.CloseConnections,
			TLSHandshakeTimeout:   timeout,
			ResponseHeaderTimeout: timeout,
		}
	}

	h.TransportCache[TransportCacheKey{Scheme: scheme, Timeout: timeout}] = t
	h.transportCacheLock.Unlock()

	return t
}

// HandleAlterRequest duplicate req and sent it to alternative Backend
func (h *Handler) HandleAlterRequest(r AlternativeReq) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in ServeHTTP(alternate req) from:", r)
		}
	}()

	if response := h.handleRequest(r.req, r.timeout, r.scheme); response != nil {
		_ = response.Body.Close()
	}
}

// handleRequest sends a req and returns the response.
func (h *Handler) handleRequest(request *http.Request, timeout time.Duration, scheme string) *http.Response {
	transport := h.getTransport(scheme, timeout)
	response, err := transport.RoundTrip(request)

	if err != nil {
		log.Println("Request failed:", err)
	}

	return response
}

// SchemeAndHost parse URL into scheme and rest of endpoint
func SchemeAndHost(url string) (scheme, hostname string) {
	if strings.HasPrefix(url, "https") {
		hostname = strings.TrimPrefix(url, "https://")
		scheme = "https"
	} else {
		hostname = strings.TrimPrefix(url, "http://")
		scheme = "http"
	}

	return
}
