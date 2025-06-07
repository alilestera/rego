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

package rego_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/alilestera/rego"
)

func TestRegoSubmit(t *testing.T) {
	defer goleak.VerifyNone(t)

	r := rego.New(regoCap)
	defer r.ReleaseWait()

	var wg sync.WaitGroup
	for range tasks {
		wg.Add(1)
		r.Submit(func() {
			demoFunc()
			wg.Done()
		})
	}
	wg.Wait()
}

func TestRegoSubmitWait(t *testing.T) {
	defer goleak.VerifyNone(t)

	r := rego.New(regoCap)
	defer r.ReleaseWait()

	var num int
	for range tasks {
		r.SubmitWait(func() {
			num++
		})
	}

	assert.Equal(t, tasks, num)
}

func TestRegoReleaseCompleteWaiting(t *testing.T) {
	defer goleak.VerifyNone(t)

	r := rego.New(regoCap)
	var num int32
	var wg sync.WaitGroup

	for range tasks {
		wg.Add(1)
		go func() {
			r.Submit(func() {
				atomic.AddInt32(&num, 1)
			})
			wg.Done()
		}()
	}
	wg.Wait()
	r.ReleaseWait()

	assert.Equal(t, tasks, int(num))
}

func TestRegoReleaseIgnoreWaiting(t *testing.T) {
	defer goleak.VerifyNone(t)

	r := rego.New(regoCap)
	var num int32
	var wg sync.WaitGroup

	for range tasks {
		wg.Add(1)
		go func() {
			r.Submit(func() {
				atomic.AddInt32(&num, 1)
			})
			wg.Done()
		}()
	}
	wg.Wait()
	r.Release()

	if r.Waiting() > 0 {
		assert.NotEqual(t, tasks, int(num), "must be some tasks are ignored when waiting queue not empty")
	}
}

func TestRegoCapacity(t *testing.T) {
	defer goleak.VerifyNone(t)

	r := rego.New(0)
	defer r.Release()
	assert.Equal(t, 1, r.Cap(), "capacity should be at least one")

	r = rego.New(regoCap)
	defer r.Release()
	assert.Equal(t, regoCap, r.Cap(), "capacity should be equal to the specified value %d", regoCap)
}

func TestRegoWithMinWorkers(t *testing.T) {
	defer goleak.VerifyNone(t)

	minimum := 1000
	// Set up very short-term idle timeout to let dispatcher try to release workers.
	r := rego.New(regoCap, rego.WithMinWorkers(minimum), rego.WithIdleTimeout(time.Millisecond))
	defer r.Release()
	assert.Equal(t, minimum, r.MinWorkers(), "minimum workers should be equal to %d", minimum)

	var wg sync.WaitGroup
	for range minimum {
		wg.Add(1)
		r.Submit(func() {
			demoFunc()
			wg.Done()
		})
	}
	wg.Wait()

	// Make sure that idle timeout is exceeded and workers are tried to be released.
	time.Sleep(100 * time.Millisecond)

	assert.GreaterOrEqual(t, r.Running(), minimum, "minWorkers should greater or equal than %d", minimum)
}
