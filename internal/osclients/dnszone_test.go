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

package osclients_test

import (
	"context"
	"errors"
	"testing"

	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
)

// TestDNSZoneErrorClient verifies that the error client returns the
// configured error for every method.
func TestDNSZoneErrorClient(t *testing.T) {
	testErr := errors.New("test configured error")
	client := osclients.NewDNSZoneErrorClient(testErr)
	ctx := context.Background()

	t.Run("ListZones", func(t *testing.T) {
		var gotErr error
		for _, err := range client.ListZones(ctx, nil) {
			gotErr = err
			break
		}
		if !errors.Is(gotErr, testErr) {
			t.Errorf("ListZones: expected %v, got %v", testErr, gotErr)
		}
	})

	t.Run("CreateZone", func(t *testing.T) {
		_, err := client.CreateZone(ctx, nil)
		if !errors.Is(err, testErr) {
			t.Errorf("CreateZone: expected %v, got %v", testErr, err)
		}
	})

	t.Run("DeleteZone", func(t *testing.T) {
		err := client.DeleteZone(ctx, "id")
		if !errors.Is(err, testErr) {
			t.Errorf("DeleteZone: expected %v, got %v", testErr, err)
		}
	})

	t.Run("GetZone", func(t *testing.T) {
		_, err := client.GetZone(ctx, "id")
		if !errors.Is(err, testErr) {
			t.Errorf("GetZone: expected %v, got %v", testErr, err)
		}
	})

	t.Run("UpdateZone", func(t *testing.T) {
		_, err := client.UpdateZone(ctx, "id", nil)
		if !errors.Is(err, testErr) {
			t.Errorf("UpdateZone: expected %v, got %v", testErr, err)
		}
	})
}
