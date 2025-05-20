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

import "testing"

func TestRego(t *testing.T) {
	r := New(3)
	requests := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	respChan := make(chan string, len(requests))
	for _, v := range requests {
		r.Submit(func() {
			respChan <- v
		})
	}
	close(respChan)
	r.Stop()

	respSet := map[string]bool{}
	for v := range respChan {
		respSet[v] = true
	}

	for _, v := range requests {
		if !respSet[v] {
			t.Errorf("expected %q, got nothing", v)
		}
	}
}
