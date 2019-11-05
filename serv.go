package httptee

import (
	"io"
	"log"
	"net/http"
)

// ServeHTTP duplicates the incoming req (req) and does the req to the
// PrimaryTarget and the Alternate target discarding the Alternate response
func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if h.ForwardClientIP {
		InsertForwardedHeaders(req)
	}

	h.tee(req)

	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in ServeHTTP(production req) from:", r)
		}
	}()

	SetRequestTarget(req, h.Primary)

	if h.PrimaryHostRewrite {
		req.Host = h.Primary.Scheme
	}

	if resp := handleRequest(req, h.primaryTransport); resp != nil {
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

func (h *Handler) tee(req *http.Request) {
	if !(h.Percent == 100 || h.randomizer.Int()*100 < h.Percent) {
		return
	}

	for _, alt := range h.Alternatives {
		alterReq := DuplicateRequest(req)

		SetRequestTarget(alterReq, alt)

		if h.AlternateHostRewrite {
			alterReq.Host = alt.Host
		}

		h.jobQueue <- AlternativeReq{Handler: h, req: alterReq, timeout: h.AlternateTimeout, scheme: alt.Scheme}
	}
}
