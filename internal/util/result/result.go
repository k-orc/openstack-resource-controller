/*
Copyright 2024 The ORC Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package result

// NOTE(mdbooth): This is a stupid type without any of the useful properties of
// a proper Result type. However, I don't believe they can currently be
// implemented in Go.

type Result[T any] struct {
	ok  *T
	err error
}

func (r Result[T]) Ok() *T {
	return r.ok
}

func (r Result[T]) Err() error {
	return r.err
}

func Ok[T any](v *T) Result[T] {
	return Result[T]{ok: v}
}

func Err[T any](err error) Result[T] {
	return Result[T]{err: err}
}
