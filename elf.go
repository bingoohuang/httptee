package httptee

import (
	"log"
	"runtime"
)

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
