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
	"fmt"

	"github.com/alilestera/rego"
)

func ExampleRego_Submit() {
	r := rego.New(3)
	defer r.Release()
	requests := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	for _, v := range requests {
		r.Submit(func() {
			fmt.Println(v)
		})
	}
	// Unordered Output:
	// alpha
	// beta
	// gamma
	// delta
	// epsilon
}

func ExampleRego_SubmitWait() {
	r := rego.New(3)
	defer r.Release()
	requests := []string{"foo", "bar", "foobar", "anything", "else"}
	for _, v := range requests {
		r.SubmitWait(func() {
			fmt.Println(v)
		})
	}
	// Output:
	// foo
	// bar
	// foobar
	// anything
	// else
}
