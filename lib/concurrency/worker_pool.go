package concurrency

import "sync"

// WorkerPool is a pool of workers that manages a number of goroutines to
// run some tasks concurrently.
type WorkerPool struct {
	tasks chan func()

	stopper *sync.Once
	stop    chan int

	done chan int
}

// NewWorkerPool returns a new worker pool. This call will never block.
// This function creates a set of <thread> goroutines that watch the list of tasks
// that the worker pool has been assigned to do.
func NewWorkerPool(threads int) *WorkerPool {
	pool := &WorkerPool{
		tasks:   make(chan func()),
		stopper: &sync.Once{},
		stop:    make(chan int, 0),
		done:    make(chan int, 0),
	}
	pool.start(threads)
	return pool
}

func (pool *WorkerPool) start(threads int) {
	var wg sync.WaitGroup
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-pool.stop:
					return
				case task, more := <-pool.tasks:
					if !more {
						return
					}
					task()
				}
			}
		}()
	}

	// Wait for all threads to return and then close the done channel.
	go func() {
		wg.Wait()
		close(pool.done)
	}()
}

// Do tells the worker pool to start executing a task. This call may block if the workers
// in the pool are all busy.
func (pool *WorkerPool) Do(fn func()) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-pool.done:
		case pool.tasks <- fn:
		}
	}()

	wg.Wait()
}

// Stop tells the worker pool to stop its goroutines, potentially
// losing some of its tasks while they are in the queue.
func (pool *WorkerPool) Stop() {
	pool.stopper.Do(func() {
		close(pool.stop)
	})
}

// Wait waits for stop or all tasks are done.
func (pool *WorkerPool) Wait() {
	close(pool.tasks)
	<-pool.done

	return
}
