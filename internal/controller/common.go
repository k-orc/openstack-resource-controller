/*
Copyright 2023.

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

package controller

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OpenStackResourceNotReadyRequeueAfter = 5 * time.Second
)

// coalesce returns the first non-empty string, or the empty string.
func coalesce(args ...string) string {
	for _, s := range args {
		if s != "" {
			return s
		}
	}
	return ""
}

func orcTag(obj client.Object) string {
	return "orc-name:" + obj.GetName()
}

// sliceContentEquals checks two slices for equivalence, discarding
// the order of the items. Accepts comparable items.
func sliceContentEquals[T comparable](slice1, slice2 []T) bool {
	return sliceContentCompare(slice1, slice2, func(item1, item2 T) bool {
		return item1 == item2
	})
}

// sliceContentCompare attemps to pair the contents of two slices with the provided
// function, discarding the order of the items. Returns true if successful.
func sliceContentCompare[T1, T2 any](slice1 []T1, slice2 []T2, comparingFunc func(T1, T2) bool) bool {
	if len(slice1) != len(slice2) {
		return false
	}

	paired := make(map[int]struct{})
Candidate:
	for i := range slice1 {
		for j := range slice2 {
			if _, ok := paired[j]; !ok && comparingFunc(slice1[i], slice2[j]) {
				paired[j] = struct{}{}
				continue Candidate
			}
		}
		return false
	}
	return true
}
