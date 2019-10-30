package httptee

import (
	"math/rand"
	"net/http"
	"time"
)

// Handler contains the address of the main PrimaryTarget and the one for the Host target
type Handler struct {
	Primary      Backend
	Alternatives []Backend

	ForwardClientIP      bool
	AlternateHostRewrite bool
	PrimaryHostRewrite   bool
	CloseConnections     bool

	Percent          int
	AlternateTimeout time.Duration
	PrimaryTimeout   time.Duration

	alterRequestChan chan AlternativeReq
	randomizer       rand.Rand

	primaryTransport *http.Transport
	alterTransport   *http.Transport
}

// Backend represents the backend server.
type Backend struct {
	Host   string
	Scheme string
}

// Backends represents array of backend servers.
type Backends []Backend

func (i *Backends) String() string {
	return "my backends representation(n/a)"
}

// Set sets backends
func (i *Backends) Set(value string) error {
	*i = append(*i, schemeAndHost(value))

	return nil
}

// AlternativeReq represents the alternative request.
type AlternativeReq struct {
	req     *http.Request
	timeout time.Duration
	scheme  string
}
