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

package v1alpha1

// +kubebuilder:validation:Format:=uuid
// +kubebuilder:validation:MaxLength:=36
type UUID string

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=1024
type OpenStackName string

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=1024
type OpenStackDescription string

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=253
type ORCNameRef string