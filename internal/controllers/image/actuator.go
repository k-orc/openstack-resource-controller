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

package image

import (
	"context"
	"fmt"
	"iter"
	"reflect"
	"slices"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

type (
	osResourceT = images.Image

	createResourceActuator = generic.CreateResourceActuator2[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = generic.DeleteResourceActuator2[orcObjectPT, orcObjectT, osResourceT]
	imageIterator          = iter.Seq2[*osResourceT, error]
)

type imageActuator struct {
	osClient osclients.ImageClient
}

func newActuator(ctx context.Context, controller generic.ResourceController, orcObject *orcv1alpha1.Image) (imageActuator, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return imageActuator{}, err
	}
	osClient, err := clientScope.NewImageClient()
	if err != nil {
		return imageActuator{}, err
	}

	return imageActuator{
		osClient: osClient,
	}, nil
}

var _ createResourceActuator = imageActuator{}
var _ deleteResourceActuator = imageActuator{}

func (imageActuator) GetResourceID(osResource *images.Image) string {
	return osResource.ID
}

func (actuator imageActuator) GetOSResourceByID(ctx context.Context, id string) (*images.Image, error) {
	return actuator.osClient.GetImage(ctx, id)
}

func (actuator imageActuator) ListOSResourcesForAdoption(ctx context.Context, obj orcObjectPT) (imageIterator, bool) {
	if obj.Spec.Resource == nil {
		return nil, false
	}

	listOpts := images.ListOpts{Name: string(getResourceName(obj))}
	return actuator.osClient.ListImages(ctx, listOpts), true
}

func (actuator imageActuator) ListOSResourcesForImport(ctx context.Context, filter filterT) imageIterator {
	listOpts := images.ListOpts{Name: string(ptr.Deref(filter.Name, ""))}
	return actuator.osClient.ListImages(ctx, listOpts)
}

func (actuator imageActuator) CreateResource(ctx context.Context, obj *orcv1alpha1.Image) ([]generic.WaitingOnEvent, *images.Image, error) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	if resource.Content == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource.content is not set")
	}

	tags := make([]string, len(resource.Tags))
	for i := range resource.Tags {
		tags[i] = string(resource.Tags[i])
	}
	// Sort tags before creation to simplify comparisons
	slices.Sort(tags)

	var minDisk, minMemory int
	properties := resource.Properties
	additionalProperties := map[string]string{}
	if properties != nil {
		if properties.MinDiskGB != nil {
			minDisk = int(*properties.MinDiskGB)
		}
		if properties.MinMemoryMB != nil {
			minMemory = int(*properties.MinMemoryMB)
		}

		if err := glancePropertiesFromStruct(properties.Hardware, additionalProperties); err != nil {
			return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "programming error", err)
		}
	}

	var visibility *images.ImageVisibility
	if resource.Visibility != nil {
		visibility = ptr.To(images.ImageVisibility(*resource.Visibility))
	}

	image, err := actuator.osClient.CreateImage(ctx, &images.CreateOpts{
		Name:            string(getResourceName(obj)),
		Visibility:      visibility,
		Tags:            tags,
		ContainerFormat: string(resource.Content.ContainerFormat),
		DiskFormat:      (string)(resource.Content.DiskFormat),
		MinDisk:         minDisk,
		MinRAM:          minMemory,
		Protected:       resource.Protected,
		Properties:      additionalProperties,
	})

	// We should require the spec to be updated before retrying a create which returned a conflict
	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating image: "+err.Error(), err)
	}

	return nil, image, err
}

func (actuator imageActuator) DeleteResource(ctx context.Context, _ orcObjectPT, osResource *images.Image) ([]generic.WaitingOnEvent, error) {
	return nil, actuator.osClient.DeleteImage(ctx, osResource.ID)
}

// glancePropertiesFromStruct populates a properties struct using field values and glance tags defined on the given struct
// glance tags are defined in the API.
func glancePropertiesFromStruct(propStruct interface{}, properties map[string]string) error {
	sp := reflect.ValueOf(propStruct)
	if sp.Kind() != reflect.Pointer {
		return fmt.Errorf("glancePropertiesFromStruct expects pointer to struct, got %T", propStruct)
	}
	if sp.IsZero() {
		return nil
	}

	s := sp.Elem()
	st := s.Type()
	if st.Kind() != reflect.Struct {
		return fmt.Errorf("glancePropertiesFromStruct expects pointer to struct, got %T", propStruct)
	}

	for i := range st.NumField() {
		field := st.Field(i)
		glanceTag, ok := field.Tag.Lookup(orcv1alpha1.GlanceTag)
		if !ok {
			panic(fmt.Errorf("glance tag not defined for field %s on struct %T", field.Name, st.Name))
		}

		value := s.Field(i)
		if value.Kind() == reflect.Pointer {
			if value.IsZero() {
				continue
			}
			value = value.Elem()
		}

		// Gophercloud takes only strings, but values may not be
		// strings. Value.String() prints semantic information for
		// non-strings, but Sprintf does what we want.
		properties[glanceTag] = fmt.Sprintf("%v", value)
	}

	return nil
}
