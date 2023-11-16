/*
Copyright 2023 Red Hat

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

package util

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OpenStackFieldOwner = "openstack.gopherkube.dev"
)

func Apply(ctx context.Context, kclient client.Client, obj client.Object, patch client.Object, excludePaths ...string) error {
	data, err := getFilteredObject(patch, excludePaths)
	if err != nil {
		return err
	}

	err = kclient.Patch(ctx, obj, client.RawPatch(types.ApplyPatchType, data), client.ForceOwnership, client.FieldOwner(OpenStackFieldOwner))
	if err != nil {
		return fmt.Errorf("failed to apply patch %s: %w", string(data), err)
	}
	return err
}

func ApplyStatus(ctx context.Context, kclient client.Client, obj client.Object, patch client.Object, excludePaths ...string) error {
	data, err := getFilteredObject(patch, excludePaths)
	if err != nil {
		return err
	}

	err = kclient.Status().Patch(ctx, obj, client.RawPatch(types.ApplyPatchType, data), client.ForceOwnership, client.FieldOwner(OpenStackFieldOwner))
	if err != nil {
		return fmt.Errorf("failed to apply status patch %s: %w", string(data), err)
	}
	return err
}

func getFilteredObject(obj client.Object, excludePaths []string) ([]byte, error) {
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	// Remove fields matching any excludePaths
	for _, path := range excludePaths {
		fields := strings.Split(path, ".")
		removePath(unstructured, fields)
	}

	return json.Marshal(unstructured)
}

func removePath(obj map[string]interface{}, path []string) {
	if len(path) == 1 {
		delete(obj, path[0])
		return
	}

	field, ok := obj[path[0]]
	if !ok {
		// The path does not exist, so there is nothing to remove.
		return
	}

	// This will panic if the intermediate path exists, but is not a map.
	// This is appropriate: paths should be hard-coded, so this would be a
	// programming error.
	removePath(field.(map[string]interface{}), path[1:])
}
