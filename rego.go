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
	maxWorkers int
	running    int32

	submit chan func()
	task   chan func()

	waitingQueue *deque.Deque[func()]
	waiting      int32

	allDone chan struct{}
	pause   chan struct{}

	stopLock sync.Mutex
	stopOnce sync.Once
	stopped  bool
}

func New(maxWorkers int) *Rego {
	// Require at least one worker.
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	r := &Rego{
		maxWorkers:   maxWorkers,
		submit:       make(chan func()),
		task:         make(chan func()),
		waitingQueue: &deque.Deque[func()]{},
		allDone:      make(chan struct{}),
		pause:        make(chan struct{}),
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

func (r *Rego) Running() int {
	return int(atomic.LoadInt32(&r.running))
}

func (r *Rego) Waiting() int {
	return int(atomic.LoadInt32(&r.waiting))
}

func (r *Rego) Pause(ctx context.Context) {
	r.stopLock.Lock()
	defer r.stopLock.Unlock()
	if r.stopped {
		return
	}

	var wg sync.WaitGroup
	wg.Add(r.maxWorkers)
	for range r.maxWorkers {
		r.Submit(func() {
			wg.Done()
			select {
			case <-ctx.Done():
			case <-r.pause:
			}
		})
	}
	wg.Wait()
}

func (r *Rego) Stop() {
	r.stopOnce.Do(func() {
		r.stopLock.Lock()
		r.stopped = true
		r.stopLock.Unlock()

		close(r.submit)
		close(r.pause)
		<-r.allDone
		close(r.task)
	})
}

func (r *Rego) dispatch() {
	var wg sync.WaitGroup
	timeout := time.NewTimer(idleTimeout)
	defer timeout.Stop()

loop:
	for {
		if r.Running() < r.maxWorkers {
			wg.Add(1)
			go r.worker(r.task, &wg)
		}
		// If there are tasks in the waiting queue, the number of workers has
		// reached maxWorkers. These tasks are executed first, and incoming
		// tasks are pushed to the waiting queue.
		if r.Waiting() > 0 {
			r.processWaiting()
			select {
			case fn, ok := <-r.submit:
				if !ok {
					break loop
				}
				r.enqueueWaiting(fn)
			default:
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
			if r.Running() > 0 {
				r.releaseWorker(false)
			}
			timeout.Reset(idleTimeout)
		}
	}

	for range r.Waiting() {
		r.processWaiting()
	}

	// Stop all workers
	for range r.Running() {
		r.releaseWorker(true)
	}
	wg.Wait()

	close(r.allDone)
}

func (r *Rego) addRunning(delta int32) {
	atomic.AddInt32(&r.running, delta)
}

func (r *Rego) addWaiting(delta int32) {
	atomic.AddInt32(&r.waiting, delta)
}

func (r *Rego) worker(task <-chan func(), wg *sync.WaitGroup) {
	defer func() {
		r.addRunning(-1)
		wg.Done()
	}()
	r.addRunning(1)
	for fn := range task {
		if fn == nil {
			return
		}
		fn()
	}
}

func (r *Rego) releaseWorker(force bool) {
	select {
	case r.task <- nil:
	default:
		if force {
			r.task <- nil
		}
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
