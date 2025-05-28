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
	"time"

	"github.com/gammazero/deque"
)

const (
	idleTimeout = 2 * time.Second
)

type Rego struct {
	workerCache sync.Pool
	maxWorkers  int
	workerCount int
	workerChan  chan func()
	taskChan    chan func()
	waiting     *deque.Deque[func()]

	shutdownChan chan struct{}
	pauseChan    chan struct{}
	stopLock     sync.Mutex
	stopOnce     sync.Once
	stopped      bool
}

func New(maxWorkers int) *Rego {
	// Require at least one worker.
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	r := &Rego{
		maxWorkers:   maxWorkers,
		workerChan:   make(chan func()),
		taskChan:     make(chan func()),
		waiting:      &deque.Deque[func()]{},
		shutdownChan: make(chan struct{}),
		pauseChan:    make(chan struct{}),
	}

	go r.dispatch()

	return r
}

func (r *Rego) Submit(task func()) {
	if task != nil {
		r.taskChan <- task
	}
}

func (r *Rego) SubmitWait(task func()) {
	if task == nil {
		return
	}
	done := make(chan struct{})
	r.Submit(func() {
		task()
		close(done)
	})
	<-done
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
			case <-r.pauseChan:
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

		close(r.taskChan)
		close(r.pauseChan)
		<-r.shutdownChan
		close(r.workerChan)

		r.waiting = nil
	})
}

func (r *Rego) dispatch() {
	var wg sync.WaitGroup
	var idle bool
	timeout := time.NewTimer(idleTimeout)
	defer timeout.Stop()

Loop:
	for {
		// If there are tasks in the waiting queue, the number of workers has
		// reached maxWorkers. These tasks are executed first, and incoming
		// tasks are pushed to the waiting queue.
		if r.waiting.Len() > 0 {
			select {
			case task, ok := <-r.taskChan:
				if !ok {
					break Loop
				}
				r.waiting.PushBack(task)
			default:
				r.workerChan <- r.waiting.PopFront()
			}
			continue
		}

		select {
		case task, ok := <-r.taskChan:
			if !ok {
				break Loop
			}
			// Execute the task or add it to the waiting queue.
			select {
			case r.workerChan <- task:
			default:
				// If workerCount is less than maxWorkers, create a new worker.
				if r.workerCount < r.maxWorkers {
					// worker returns at latest when workerChan is closed.
					wg.Add(1)
					go func() {
						defer wg.Done()
						worker(r.workerChan)
					}()
					r.workerCount++

					r.workerChan <- task
				} else {
					r.waiting.PushBack(task)
				}
			}
			idle = false
		case <-timeout.C:
			if idle && r.workerCount > 0 {
				if r.tryKillIdleWorker() {
					r.workerCount--
				}
			}
			idle = true
			timeout.Reset(idleTimeout)
		}
	}

	for r.waiting.Len() > 0 {
		r.workerChan <- r.waiting.PopFront()
	}

	// Stop all workers
	for r.workerCount > 0 {
		r.workerChan <- nil
		r.workerCount--
	}
	wg.Wait()

	close(r.shutdownChan)
}

func (r *Rego) tryKillIdleWorker() bool {
	select {
	case r.workerChan <- nil:
		return true
	default:
		return false
	}
}

func worker(workerChan <-chan func()) {
	for task := range workerChan {
		if task == nil {
			return
		}
		task()
	}
}
