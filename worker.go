package httptee

import "log"

// Job represents the job to be run
type Job interface {
	Do() error
}

// JobPool alias a channel for Job
type JobPool chan Job

// Worker represents the worker that executes the job
type Worker struct {
	WorkerPool chan JobPool
	JobChannel JobPool
	quit       chan bool
}

// NewWorker creates a new Worker
func NewWorker(workerPool chan JobPool) Worker {
	return Worker{
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		quit:       make(chan bool)}
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w Worker) Start() {
	go func() {
		for {
			// register the current worker into the worker queue.
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				// we have received a work request.
				if err := job.Do(); err != nil {
					log.Printf("Error job.Do() : %v\n", err.Error())
				}

			case <-w.quit:
				return
			}
		}
	}()
}

// Stop signals the worker to stop listening for work requests.
func (w Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}

// Dispatcher dispatches the job to worker in the pool.
type Dispatcher struct {
	// A pool of workers channels that are registered with the dispatcher
	jobPool    JobPool
	workerPool chan JobPool
	maxWorkers int
}

// NewDispatcher create a new Dispatcher.
func NewDispatcher(jobPool JobPool, maxWorkers int) *Dispatcher {
	return &Dispatcher{
		workerPool: make(chan JobPool, maxWorkers),
		jobPool:    jobPool,
		maxWorkers: maxWorkers,
	}
}

// Run runs the dispatching.
func (d *Dispatcher) Run() {
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewWorker(d.workerPool)
		worker.Start()
	}

	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for job := range d.jobPool {
		// a job request has been received
		go func(job Job) {
			// try to obtain a worker job channel that is available.
			// this will block until a worker is idle
			jobChannel := <-d.workerPool

			// dispatch the job to the worker job channel
			jobChannel <- job
		}(job)
	}
}
