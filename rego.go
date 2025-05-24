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
	"container/list"
	"sync"
	"time"
)

const (
	idleTimeout = 2 * time.Second
)

type Rego struct {
	maxWorkers  int
	workerCount int
	workerChan  chan func()
	taskChan    chan func()
	waiting     *list.List

	shutdownChan chan struct{}
	stopOnce     sync.Once
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
		waiting:      list.New(),
		shutdownChan: make(chan struct{}),
	}

	go r.dispatch()

	return r
}

func (r *Rego) Submit(task func()) {
	if task != nil {
		r.taskChan <- task
	}
}

func (r *Rego) Stop() {
	r.stopOnce.Do(func() {
		close(r.taskChan)
		<-r.shutdownChan
		close(r.workerChan)
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
			case r.workerChan <- r.waiting.Front().Value.(func()):
				r.waiting.Remove(r.waiting.Front())
			}
			continue
		}

		select {
		case task, ok := <-r.taskChan:
			if !ok {
				break Loop
			}
			// Execute the task or add it to the waiting queue
			select {
			case r.workerChan <- task:
			default:
				// If workerCount is less than maxWorkers, create a new worker
				if r.workerCount < r.maxWorkers {
					wg.Add(1)
					go worker(task, r.workerChan, &wg)
					r.workerCount++
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
		r.workerChan <- r.waiting.Remove(r.waiting.Front()).(func())
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

func worker(task func(), workerChan chan func(), wg *sync.WaitGroup) {
	defer wg.Done()
	// If the task is nil, exit the worker
	for task != nil {
		task()
		task = <-workerChan
	}
}
