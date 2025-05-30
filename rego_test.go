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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/alilestera/rego"
)

func TestSubmit(t *testing.T) {
	defer goleak.VerifyNone(t)
	r := rego.New(3)

	requests := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	respChan := make(chan string, len(requests))
	for _, req := range requests {
		r.Submit(func() {
			time.Sleep(100 * time.Millisecond)
			respChan <- req
		})
	}
	r.Release()
	close(respChan)

	respSet := make(map[string]bool)
	for resp := range respChan {
		respSet[resp] = true
	}

	assert.Equal(t, len(requests), len(respSet))
	for _, req := range requests {
		if !respSet[req] {
			assert.Truef(t, respSet[req], "expected response %s, but not found", req)
		}
	}
}

func TestRegoMaxWorkers(t *testing.T) {
	maxWorkers := 5
	r := rego.New(maxWorkers)
	defer r.Release()

	for range maxWorkers * 10 {
		r.Submit(func() {
			running := r.Running()
			if running > maxWorkers {
				t.Errorf("running workers exceeded maxWorkers: %d > %d", running, maxWorkers)
			}
			time.Sleep(10 * time.Millisecond)
		})
	}
}
