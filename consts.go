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

import "time"

const (
	// DefaultMinWorkers is the default minimum number of workers to keep alive.
	DefaultMinWorkers = 0
	// DefaultIdleTimeout is the default timeout for idle workers.
	DefaultIdleTimeout = 2 * time.Second
)

const (
	// Opened represents that the Rego is opened.
	Opened = iota
	// Closed represents that the Rego is closed.
	Closed
)
