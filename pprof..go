package httptee

import (
	"flag"
	"log"
	"net/http"
)

// PprofAddrFlag defines the pprof listening address.
// see https://golang.org/pkg/net/http/pprof/
func PprofAddrFlag() *string {
	return flag.String("pprof", "",
		"pprof address to listen on, eg. -pprof :6060")
}

// StartPprof starts the pprof http service.
func StartPprof(pprofAddr string) {
	if pprofAddr == "" {
		return
	}

	log.Println("Starting pprof at: http://" + pprofAddr + "/debug/pprof")

	go func() {
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			log.Panic("error", err)
		}
	}()
}
