package labels

import (
	"strings"
)

// ReplacePrefixed returns a copy of originalLabels after:
//   - removing all keys with the given prefix;
//   - merging newLabels.
//
// Inputs are not modified. The returned labels map is guaranteed to be
// non-nil. Updated is true if the elements of labels do not match the elements
// of originalLabels.
func ReplacePrefixed(prefix string, originalLabels map[string]string, newLabels map[string]string) (labels map[string]string, updated bool) {
	labels = make(map[string]string)

	for k := range originalLabels {
		if currentPrefix, _, hasPrefix := strings.Cut(k, "/"); hasPrefix && currentPrefix != prefix || !hasPrefix && prefix != "" {
			labels[k] = originalLabels[k]
		}
	}

	for k := range newLabels {
		labels[k] = newLabels[k]
	}

	if len(originalLabels) != len(labels) {
		return labels, true
	}

	for k := range labels {
		if v, ok := originalLabels[k]; !ok || v != labels[k] {
			return labels, true
		}
	}

	return labels, false
}
