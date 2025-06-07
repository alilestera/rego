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
	"testing"
	"time"

	"github.com/alilestera/rego"
)

var runTimes = 100000

var (
	shortTerm = 10
	regoCap   = 10000
	tasks     = 100000
)

// demoFunc is used to simulate a task that takes a certain amount of time to complete.
func demoFunc() {
	time.Sleep(time.Duration(shortTerm) * time.Millisecond)
}

func BenchmarkRunGoroutines(b *testing.B) {
	var wg sync.WaitGroup
	for range b.N {
		wg.Add(runTimes)
		for range runTimes {
			go func() {
				demoFunc()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkRunRego(b *testing.B) {
	var wg sync.WaitGroup
	r := rego.New(regoCap)
	defer r.ReleaseWait()

	b.ResetTimer()
	for range b.N {
		wg.Add(runTimes)
		for range runTimes {
			r.Submit(func() {
				demoFunc()
				wg.Done()
			})
		}
		wg.Wait()
	}
}
