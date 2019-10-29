package httptee

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"runtime"
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

func (h Handler) getTransport(scheme string, timeout time.Duration) *http.Transport {
	if scheme == "https" {
		return &http.Transport{
			Dial: (&net.Dialer{ // go1.8 deprecated: Use DialContext instead
				Timeout:   timeout,
				KeepAlive: 10 * timeout,
			}).Dial,
			DisableKeepAlives:     h.CloseConnections,
			TLSHandshakeTimeout:   timeout,
			ResponseHeaderTimeout: timeout,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, // nolint
		}
	}

	return &http.Transport{
		Dial: (&net.Dialer{ // go1.8 deprecated: Use DialContext instead
			Timeout:   timeout,
			KeepAlive: 10 * timeout,
		}).Dial,
		DisableKeepAlives:     h.CloseConnections,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
	}
}

// HandleAlterRequest duplicate req and sent it to alternative Backend
func (h Handler) HandleAlterRequest(r AlternativeReq) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in ServeHTTP(alternate req) from:", r)
		}
	}()

	if response := h.handleRequest(r.req, r.timeout, r.scheme); response != nil {
		response.Body.Close()
	}
}

// Sends a req and returns the response.
func (h Handler) handleRequest(request *http.Request, timeout time.Duration, scheme string) *http.Response {
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

// Handler contains the address of the main PrimaryTarget and the one for the Endpoint target
type Handler struct {
	PrimaryTarget string
	TargetScheme  string
	Alternatives  []Backend
	Randomizer    rand.Rand

	ForwardClientIP      bool
	AlternateHostRewrite bool
	PrimaryHostRewrite   bool
	CloseConnections     bool
	Percent              int
	AlternateTimeout     time.Duration
	PrimaryTimeout       time.Duration
	AlterRequestChan     chan AlternativeReq
}

// Backend represents the backend server.
type Backend struct {
	Endpoint string
	Scheme   string
}

// Backends represents array of backend servers.
type Backends []Backend

func (i *Backends) String() string {
	return "my backends representation(n/a)"
}

// Set sets backends
func (i *Backends) Set(value string) error {
	scheme, endpoint := SchemeAndHost(value)
	altServer := Backend{Scheme: scheme, Endpoint: endpoint}
	*i = append(*i, altServer)

	return nil
}

// SetSchemes set schemes.
func (h *Handler) SetSchemes() {
	h.TargetScheme, h.PrimaryTarget = SchemeAndHost(h.PrimaryTarget)
}

// AlternativeReq represents the alternative request.
type AlternativeReq struct {
	req     *http.Request
	timeout time.Duration
	scheme  string
}

// ServeHTTP duplicates the incoming req (req) and does the req to the
// PrimaryTarget and the Alternate target discarding the Alternate response
func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if h.ForwardClientIP {
		updateForwardedHeaders(req)
	}

	if h.Percent == 100 || h.Randomizer.Int()*100 < h.Percent {
		for _, alt := range h.Alternatives {
			alterReq := DuplicateRequest(req)

			setRequestTarget(alterReq, alt.Endpoint, alt.Scheme)

			if h.AlternateHostRewrite {
				alterReq.Host = alt.Endpoint
			}

			h.AlterRequestChan <- AlternativeReq{req: alterReq, timeout: h.AlternateTimeout, scheme: alt.Scheme}
		}
	}

	productionRequest := req

	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in ServeHTTP(production req) from:", r)
		}
	}()

	setRequestTarget(productionRequest, h.PrimaryTarget, h.TargetScheme)

	if h.PrimaryHostRewrite {
		productionRequest.Host = h.PrimaryTarget
	}

	resp := h.handleRequest(productionRequest, h.PrimaryTimeout, h.TargetScheme)

	if resp != nil {
		defer resp.Body.Close()

		// Forward response headers.
		for k, v := range resp.Header {
			w.Header()[k] = v
		}

		w.WriteHeader(resp.StatusCode)

		// Forward response body.
		_, _ = io.Copy(w, resp.Body)
	}
}

// StartWorkers start number of workers to call infiniteFunc.
func StartWorkers(workers int, infiniteFunc func()) {
	if workers == 0 {
		workers = runtime.NumCPU()
	}

	for i := 0; i < workers; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered in worker from:", r)
				}
			}()

			infiniteFunc()
		}()
	}
}

// DuplicateRequest duplicate http req
func DuplicateRequest(request *http.Request) (dup *http.Request) {
	var bodyBytes []byte
	if request.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(request.Body)
	}

	request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	dup = &http.Request{
		Method:        request.Method,
		URL:           request.URL,
		Proto:         request.Proto,
		ProtoMajor:    request.ProtoMajor,
		ProtoMinor:    request.ProtoMinor,
		Header:        request.Header,
		Body:          ioutil.NopCloser(bytes.NewBuffer(bodyBytes)),
		Host:          request.Host,
		ContentLength: request.ContentLength,
		Close:         true,
	}

	return
}

func updateForwardedHeaders(request *http.Request) {
	var remoteIP string

	if positionOfColon := strings.LastIndex(request.RemoteAddr, ":"); positionOfColon != -1 {
		remoteIP = request.RemoteAddr[:positionOfColon]
	} else {
		log.Printf("The default format of req.RemoteAddr should be IP:Port but was %s\n", remoteIP)
		remoteIP = request.RemoteAddr
	}

	insertOrExtendForwardedHeader(request, remoteIP)
	insertOrExtendXFFHeader(request, remoteIP)
}

const xffHeader = "X-Forwarded-For"

func insertOrExtendXFFHeader(request *http.Request, remoteIP string) {
	header := request.Header.Get(xffHeader)

	if header != "" { // extend
		request.Header.Set(xffHeader, header+", "+remoteIP)
	} else { // insert
		request.Header.Set(xffHeader, remoteIP)
	}
}

const forwardedHeader = "Forwarded"

// Implementation according to rfc7239
func insertOrExtendForwardedHeader(request *http.Request, remoteIP string) {
	extension := "for=" + remoteIP
	header := request.Header.Get(forwardedHeader)

	if header != "" { // extend
		request.Header.Set(forwardedHeader, header+", "+extension)
	} else { // insert
		request.Header.Set(forwardedHeader, extension)
	}
}
