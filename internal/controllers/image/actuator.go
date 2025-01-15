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
	"reflect"
	"slices"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

type imageActuator struct {
	*orcv1alpha1.Image
	osClient   osclients.ImageClient
	controller generic.ResourceController
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
		Image:      orcObject,
		osClient:   osClient,
		controller: controller,
	}, nil
}

var _ generic.CreateResourceActuator[*images.Image] = imageActuator{}
var _ generic.DeleteResourceActuator[*images.Image] = imageActuator{}

func (obj imageActuator) GetObject() client.Object {
	return obj.Image
}

func (obj imageActuator) GetController() generic.ResourceController {
	return obj.controller
}

func (obj imageActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return obj.Spec.ManagementPolicy
}

func (obj imageActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return obj.Spec.ManagedOptions
}

func (obj imageActuator) GetResourceID(osResource *images.Image) string {
	return osResource.ID
}

func (obj imageActuator) GetStatusID() *string {
	return obj.Status.ID
}

func (obj imageActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *images.Image, error) {
	if obj.Status.ID == nil {
		return false, nil, nil
	}

	image, err := obj.osClient.GetImage(*obj.Status.ID)
	return true, image, err
}

func (obj imageActuator) GetOSResourceBySpec(ctx context.Context) (*images.Image, error) {
	if obj.Spec.Resource == nil {
		return nil, nil
	}
	listOpts := listOptsFromCreation(obj.Image)
	image, err := getGlanceImageFromList(ctx, listOpts, obj.osClient)
	return image, err
}

func (obj imageActuator) GetOSResourceByImportID(ctx context.Context) (bool, *images.Image, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.ID == nil {
		return false, nil, nil
	}

	image, err := obj.osClient.GetImage(*obj.Spec.Import.ID)
	return true, image, err
}

func (obj imageActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *images.Image, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	listOpts := listOptsFromImportFilter(obj.Spec.Import.Filter)
	image, err := getGlanceImageFromList(ctx, listOpts, obj.osClient)
	return true, image, err
}

func (obj imageActuator) CreateResource(ctx context.Context) ([]generic.WaitingOnEvent, *images.Image, error) {
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
			minDisk = *properties.MinDiskGB
		}
		if properties.MinMemoryMB != nil {
			minMemory = *properties.MinMemoryMB
		}

		if err := glancePropertiesFromStruct(properties.Hardware, additionalProperties); err != nil {
			return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "programming error", err)
		}
	}

	var visibility *images.ImageVisibility
	if resource.Visibility != nil {
		visibility = ptr.To(images.ImageVisibility(*resource.Visibility))
	}

	image, err := obj.osClient.CreateImage(ctx, &images.CreateOpts{
		Name:            string(getResourceName(obj.Image)),
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

func (obj imageActuator) DeleteResource(ctx context.Context, osResource *images.Image) ([]generic.WaitingOnEvent, error) {
	return nil, obj.osClient.DeleteImage(ctx, osResource.ID)
}

// getResourceName returns the name of the glance image we should use.
func getResourceName(orcImage *orcv1alpha1.Image) orcv1alpha1.OpenStackName {
	if orcImage.Spec.Resource.Name != nil {
		return *orcImage.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcImage.Name)
}

func listOptsFromImportFilter(filter *orcv1alpha1.ImageFilter) images.ListOptsBuilder {
	return images.ListOpts{Name: ptr.Deref(filter.Name, "")}
}

// listOptsFromCreation returns a listOpts which will return the image which
// would have been created from the current spec and hopefully no other image.
// Its purpose is to automatically adopt an image that we created but failed to
// write to status.id.
func listOptsFromCreation(orcImage *orcv1alpha1.Image) images.ListOptsBuilder {
	return images.ListOpts{Name: string(getResourceName(orcImage))}
}

func getGlanceImageFromList(_ context.Context, listOpts images.ListOptsBuilder, imageClient osclients.ImageClient) (*images.Image, error) {
	glanceImages, err := imageClient.ListImages(listOpts)
	if err != nil {
		return nil, err
	}

	if len(glanceImages) == 1 {
		return &glanceImages[0], nil
	}

	// No image found
	if len(glanceImages) == 0 {
		return nil, nil
	}

	// Multiple images found
	return nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, fmt.Sprintf("Expected to find exactly one image to import. Found %d", len(glanceImages)))
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
