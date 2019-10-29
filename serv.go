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

	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in ServeHTTP(production req) from:", r)
		}
	}()

	setRequestTarget(req, h.PrimaryTarget, h.TargetScheme)

	if h.PrimaryHostRewrite {
		req.Host = h.PrimaryTarget
	}

	if resp := h.handleRequest(req, h.PrimaryTimeout, h.TargetScheme); resp != nil {
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
