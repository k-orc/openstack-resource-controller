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

package v1alpha1

// ImageMembershipStatus is the status of an image member.
// +kubebuilder:validation:Enum:=accepted;rejected
type ImageMembershipStatus string

const (
	ImageMembershipStatusAccepted ImageMembershipStatus = "accepted"
	ImageMembershipStatusRejected ImageMembershipStatus = "rejected"
)

// ImageMemberResourceSpec contains the desired state of the resource.
type ImageMemberResourceSpec struct {
	// imageRef is a reference to the ORC Image shared with the project of
	// this object's credentials. The Image's members must include that
	// project.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="imageRef is immutable"
	// +orc:kustomize:ref=Image
	ImageRef KubernetesNameRef `json:"imageRef,omitempty"`

	// status is the membership status to set. accepted makes the image
	// appear in the project's image list. Deleting the object resets the
	// status to pending; only the image owner can remove the member.
	// +kubebuilder:default:=accepted
	// +optional
	Status *ImageMembershipStatus `json:"status,omitempty"`
}

// ImageMemberFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type ImageMemberFilter struct {
	// imageRef is a reference to the ORC Image shared with the project of
	// this object's credentials.
	// +optional
	// +orc:kustomize:ref=Image
	ImageRef *KubernetesNameRef `json:"imageRef,omitempty"`
}

// ImageMemberResourceStatus represents the observed state of the resource.
type ImageMemberResourceStatus struct {
	// imageID is the ID of the shared image.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ImageID string `json:"imageID,omitempty"`

	// memberID is the ID of the project the image is shared with.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	MemberID string `json:"memberID,omitempty"`

	// status is the membership status: pending, accepted or rejected.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Status string `json:"status,omitempty"`
}
