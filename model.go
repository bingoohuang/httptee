package httptee

import (
	"math/rand"
	"net/http"
	"sync"
	"time"
)

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

	transportCacheLock sync.RWMutex
	TransportCache     map[TransportCacheKey]*http.Transport
}

// TransportCacheKey means key structure of TransportCacheKey
type TransportCacheKey struct {
	Scheme  string
	Timeout time.Duration
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
