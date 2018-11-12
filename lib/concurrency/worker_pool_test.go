package concurrency_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/uber/makisu/lib/concurrency"
	"github.com/stretchr/testify/require"
)

func TestWorkerPool(t *testing.T) {
	require := require.New(t)
	pool := concurrency.NewWorkerPool(100)
	count := int32(0)
	for i := 0; i < 100; i++ {
		pool.Do(func() {
			time.Sleep(1 * time.Millisecond)
			atomic.AddInt32(&count, 1)
		})
	}
	pool.Wait()
	require.Equal(int32(100), count)
}

func TestWorkerPoolStop(t *testing.T) {
	require := require.New(t)
	pool := concurrency.NewWorkerPool(5)
	count := int32(0)
	for i := 0; i < 5; i++ {
		pool.Do(func() {
			time.Sleep(1 * time.Millisecond)
			atomic.AddInt32(&count, 1)
		})
	}

	var wg sync.WaitGroup
	wg.Add(1)
	pool.Do(func() {
		defer wg.Done()
		atomic.AddInt32(&count, 1)
		pool.Stop()
	})
	wg.Wait()

	// Some future tasks will be executed after stop is called.
	for i := 6; i < 100; i++ {
		pool.Do(func() {
			time.Sleep(1 * time.Millisecond)
			atomic.AddInt32(&count, 1)
		})
	}

	pool.Wait()
	require.True(count >= 6)
}
