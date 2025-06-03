/*
 * Copyright 2025 alilestera
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rego

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gammazero/deque"
)

const (
	idleTimeout = 2 * time.Second
)

type Rego struct {
	capacity int
	running  int32

	submit       chan func()
	task         chan func()
	waitingQueue *deque.Deque[func()]
	waiting      int32

	dismiss     context.Context
	dismissFunc context.CancelFunc
	allDone     chan struct{}
	closeOnce   sync.Once

	options options
}

func New(size int, opts ...Option) *Rego {
	// Require at least one worker.
	if size < 1 {
		size = 1
	}

	o := options{
		minWorkers:  DefaultMinWorkers,
		idleTimeout: DefaultIdleTimeout,
	}
	for _, opt := range opts {
		opt(&o)
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &Rego{
		capacity:     size,
		submit:       make(chan func()),
		task:         make(chan func()),
		waitingQueue: &deque.Deque[func()]{},

		dismiss:     ctx,
		dismissFunc: cancel,
		allDone:     make(chan struct{}),

		options: o,
	}

	go r.dispatch()

	return r
}

func (r *Rego) Submit(task func()) {
	if task != nil {
		r.submit <- task
	}
}

func (r *Rego) SubmitWait(task func()) {
	if task == nil {
		return
	}
	done := make(chan struct{})
	r.submit <- func() {
		task()
		close(done)
	}
	<-done
}

func (r *Rego) Cap() int {
	return r.capacity
}

func (r *Rego) Running() int {
	return int(atomic.LoadInt32(&r.running))
}

func (r *Rego) Waiting() int {
	return int(atomic.LoadInt32(&r.waiting))
}

func (r *Rego) MinWorkers() int {
	return r.options.minWorkers
}

func (r *Rego) Release() {
	r.release(false)
}

func (r *Rego) ReleaseWait() {
	r.release(true)
}

func (r *Rego) release(wait bool) {
	r.closeOnce.Do(func() {
		if wait {
			defer r.dismissFunc()
		} else {
			r.dismissFunc()
		}
		close(r.submit)
		<-r.allDone
		close(r.task)
	})
}

func (r *Rego) dispatch() {
	var wg sync.WaitGroup
	timeout := time.NewTimer(r.options.idleTimeout)
	defer timeout.Stop()

loop:
	for {
		if r.Running() < r.Cap() {
			wg.Add(1)
			r.runWorker(r.task, &wg)
		}
		// If there are tasks in the waiting queue, the number of workers has
		// reached maxWorkers. These tasks are executed first, and incoming
		// tasks are pushed to the waiting queue.
		if r.Waiting() > 0 {
			select {
			case fn, ok := <-r.submit:
				if !ok {
					break loop
				}
				r.enqueueWaiting(fn)
			default:
				r.processWaiting()
			}
			continue
		}

		select {
		case fn, ok := <-r.submit:
			if !ok {
				break loop
			}
			// Execute the task or add it to the waiting queue.
			select {
			case r.task <- fn:
			default:
				r.enqueueWaiting(fn)
			}
		case <-timeout.C:
			if r.Running() > r.MinWorkers() {
				r.tryReleaseWorker()
			}
			timeout.Reset(r.options.idleTimeout)
		}
	}

	select {
	case <-r.dismiss.Done():
	default:
		for range r.Waiting() {
			r.processWaiting()
		}
	}
	r.releaseAllWorkers()
	wg.Wait()

	close(r.allDone)
}

func (r *Rego) addRunning(delta int32) {
	atomic.AddInt32(&r.running, delta)
}

func (r *Rego) addWaiting(delta int32) {
	atomic.AddInt32(&r.waiting, delta)
}

func (r *Rego) runWorker(task <-chan func(), wg *sync.WaitGroup) {
	r.addRunning(1)
	go func() {
		defer func() {
			r.addRunning(-1)
			wg.Done()
		}()
		for fn := range task {
			if fn == nil {
				return
			}
			fn()
		}
	}()
}

func (r *Rego) tryReleaseWorker() {
	select {
	case r.task <- nil:
	default:
	}
}

func (r *Rego) releaseAllWorkers() {
	for r.Running() > 0 {
		r.tryReleaseWorker()
	}
}

func (r *Rego) enqueueWaiting(fn func()) {
	r.waitingQueue.PushBack(fn)
	r.addWaiting(1)
}

func (r *Rego) processWaiting() {
	fn := r.waitingQueue.PopFront()
	r.task <- fn
	r.addWaiting(-1)
}
