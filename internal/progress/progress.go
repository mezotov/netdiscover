package progress

import (
	"fmt"
	"sync"
	"time"
)

type Tracker struct {
	mu     sync.Mutex
	wg     sync.WaitGroup
	counts map[string]int
	done   chan struct{}
}

func New() *Tracker {
	return &Tracker{
		counts: make(map[string]int),
		done:   make(chan struct{}),
	}
}

func (t *Tracker) Increment(source string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counts[source]++
}

func (t *Tracker) Start() {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-t.done:
				fmt.Println()
				return
			case <-ticker.C:
				t.print()
			}
		}
	}()
}

func (t *Tracker) Stop() {
	close(t.done)
	t.wg.Wait()
}

func (t *Tracker) print() {
	t.mu.Lock()
	defer t.mu.Unlock()

	fmt.Print("\rScanning ")
	for k, v := range t.counts {
		fmt.Printf("[%s:%d] ", k, v)
	}
}
