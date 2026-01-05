package worker

import "sync"

type Pool[T any] struct {
	jobs chan T
	wg   sync.WaitGroup
}

func New[T any](workers int, fn func(T)) *Pool[T] {
	p := &Pool[T]{
		jobs: make(chan T),
	}

	for i := 0; i < workers; i++ {
		go func() {
			for job := range p.jobs {
				fn(job)
				p.wg.Done()
			}
		}()
	}

	return p
}

func (p *Pool[T]) Submit(job T) {
	p.wg.Add(1)
	p.jobs <- job
}

func (p *Pool[T]) Wait() {
	p.wg.Wait()
}

func (p *Pool[T]) Close() {
	close(p.jobs)
}
