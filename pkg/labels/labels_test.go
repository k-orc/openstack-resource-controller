package labels_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/k-orc/openstack-resource-controller/pkg/labels"
)

func TestReplacePrefixed(t *testing.T) {
	type checkFunc func(map[string]string, bool) error
	check := func(fns ...checkFunc) []checkFunc { return fns }

	updated := func(want bool) checkFunc {
		return func(_ map[string]string, have bool) error {
			if want != have {
				return fmt.Errorf("expected updated %v, got %v", want, have)
			}
			return nil
		}
	}

	mapEquals := func(want map[string]string) checkFunc {
		return func(have map[string]string, _ bool) error {
			if !reflect.DeepEqual(want, have) {
				return fmt.Errorf("expected %v, got %v", want, have)
			}
			return nil
		}
	}

	for _, tc := range [...]struct {
		name                      string
		prefix                    string
		originalLabels, newLabels map[string]string
		checks                    []checkFunc
	}{
		{
			name: "nil input",
			checks: check(
				mapEquals(map[string]string{}),
				updated(false),
			),
		},
		{
			name:           "nil originalLabels",
			prefix:         "prefix",
			originalLabels: nil,
			newLabels:      map[string]string{"prefix/key": "value"},
			checks: check(
				mapEquals(map[string]string{
					"prefix/key": "value",
				}),
				updated(true),
			),
		},
		{
			name:           "empty prefix",
			prefix:         "",
			originalLabels: map[string]string{"prefix1/key1": "value1", "key2": "value2"},
			newLabels:      map[string]string{"key3": "value3"},
			checks: check(
				mapEquals(map[string]string{
					"prefix1/key1": "value1",
					"key3":         "value3",
				}),
				updated(true),
			),
		},
		{
			name:           "non-empty prefix",
			prefix:         "prefix1",
			originalLabels: map[string]string{"prefix1/key1": "value1", "prefix2/key2": "value2", "key3": "value3"},
			newLabels:      map[string]string{"prefix1/keyA": "valueA"},
			checks: check(
				mapEquals(map[string]string{
					"prefix1/keyA": "valueA",
					"prefix2/key2": "value2",
					"key3":         "value3",
				}),
				updated(true),
			),
		},
		{
			name:           "unchanged",
			prefix:         "prefix1",
			originalLabels: map[string]string{"prefix1/key1": "value1", "prefix2/key2": "value2", "key3": "value3"},
			newLabels:      map[string]string{"prefix1/key1": "value1"},
			checks: check(
				mapEquals(map[string]string{
					"prefix1/key1": "value1",
					"prefix2/key2": "value2",
					"key3":         "value3",
				}),
				updated(false),
			),
		},
		{
			name:           "replaced with empty value",
			prefix:         "prefix1",
			originalLabels: map[string]string{"prefix1/key1": "value1", "prefix2/key2": "value2", "key3": "value3"},
			newLabels:      map[string]string{"prefix1/keyA": ""},
			checks: check(
				mapEquals(map[string]string{
					"prefix1/keyA": "",
					"prefix2/key2": "value2",
					"key3":         "value3",
				}),
				updated(true),
			),
		},
		{
			name:           "has subdomain",
			prefix:         "prefix1",
			originalLabels: map[string]string{"sub.prefix1/key1": "value1", "sub.moresub.prefix1/key2": "value2"},
			newLabels:      map[string]string{"prefix1/keyA": "valueA"},
			checks: check(
				mapEquals(map[string]string{
					"prefix1/keyA": "valueA",
				}),
				updated(true),
			),
		},
		{
			name:           "preserves higher-level domains",
			prefix:         "sub.prefix1",
			originalLabels: map[string]string{"sub.prefix1/key1": "value1", "prefix1/key2": "value2"},
			newLabels:      map[string]string{"sub.prefix1/keyA": "valueA"},
			checks: check(
				mapEquals(map[string]string{
					"prefix1/key2":     "value2",
					"sub.prefix1/keyA": "valueA",
				}),
				updated(true),
			),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("unexpected panic: %v", r)
				}
			}()

			labels, updated := labels.ReplacePrefixed(tc.prefix, tc.originalLabels, tc.newLabels)
			for _, check := range tc.checks {
				if e := check(labels, updated); e != nil {
					t.Error(e)
				}
			}
		})
	}
}
