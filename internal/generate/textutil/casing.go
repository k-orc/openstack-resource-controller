/*
Copyright The ORC Authors.

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

package textutil

import (
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// CamelToSnake converts a camelCase string to snake_case.
func CamelToSnake(s string) string {
	// Add an underscore before each uppercase letter that is not at the start of the string.
	// Example: "camelCase" -> "camel_Case"
	// Example: "HTTPRequest" -> "HTTP_Request"
	re1 := regexp.MustCompile("([A-Z])([A-Z][a-z])")
	s = re1.ReplaceAllString(s, "${1}_${2}")

	// Add an underscore before each uppercase letter that is followed by a lowercase letter
	// and is not at the start of the string.
	// Example: "camel_Case" -> "camel_case" (after lowercasing)
	// Example: "HTTP_Request" -> "http_request" (after lowercasing)
	re2 := regexp.MustCompile("([a-z0-9])([A-Z])")
	s = re2.ReplaceAllString(s, "${1}_${2}")

	return strings.ToLower(s)
}

// ToCamelCase converts a string to camelCase.
//
// From https://stackoverflow.com/questions/70083837/how-to-convert-a-string-to-camelcase-in-go
func ToCamelCase(s string) string {
	// Remove all characters that are not alphanumeric or spaces or underscores
	s = regexp.MustCompile("[^a-zA-Z0-9_ ]+").ReplaceAllString(s, "")

	// Replace all underscores with spaces
	s = strings.ReplaceAll(s, "_", " ")

	// Title case s
	s = cases.Title(language.AmericanEnglish, cases.NoLower).String(s)

	// Remove all spaces
	s = strings.ReplaceAll(s, " ", "")

	// Lowercase the first letter
	if len(s) > 0 {
		s = strings.ToLower(s[:1]) + s[1:]
	}

	return s
}
