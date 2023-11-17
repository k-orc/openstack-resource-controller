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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openstackv1 "github.com/gophercloud/gopherkube/api/v1alpha1"
)

const (
	OpenStackConditionReasonReady          = "Ready"
	OpenStackConditionReadonNotReady       = "NotReady"
	OpenStackConditionReasonNoError        = "NoError"
	OpenStackConditionReasonBadCredentials = "BadCredentials"
)

// GetCondition returns the condition with the given type on the OpenStack resource, or nil if it does not exist.
func GetCondition(openStackResource openstackv1.OpenStackResourceCommonStatus, conditionType string) *metav1.Condition {
	for i := range openStackResource.OpenStackCommonStatus().Conditions {
		condition := &openStackResource.OpenStackCommonStatus().Conditions[i]
		if condition.Type == conditionType {
			return condition
		}
	}
	return nil
}

// SetCondition sets a condition on an OpenStack resource. Returns true if the condition was updated from the original resource.
func SetCondition(openStackResource, patch openstackv1.OpenStackResourceCommonStatus, condition metav1.Condition) (bool, *metav1.Condition) {
	updated := true

	origCondition := GetCondition(openStackResource, condition.Type)
	if ConditionMatches(origCondition, &condition) {
		// Copy of the original maintains LastTransitionTime
		condition = *origCondition
		updated = false
	}

	current := GetCondition(patch, condition.Type)
	if current != nil {
		*current = condition
	} else {
		patchStatus := patch.OpenStackCommonStatus()
		patchStatus.Conditions = append(patchStatus.Conditions, condition)
	}
	return updated, &condition
}

// Dependency describes an object that a resource is waiting for
type Dependency struct {
	client.ObjectKey
	Resource string
}

// String returns a string representation of the dependency
func (d Dependency) String() string {
	return d.Resource + ":" + d.Namespace + "/" + d.Name
}

func NotReadyWaiting(objects []Dependency) metav1.Condition {
	deps := make([]string, len(objects))
	for i, obj := range objects {
		deps[i] = obj.String()
	}
	message := "Waiting for the following dependencies to be ready: " + strings.Join(deps, ", ")

	return metav1.Condition{
		Type:               string(openstackv1.OpenStackConditionReady),
		Status:             metav1.ConditionFalse,
		Reason:             "WaitingForDependencies",
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
}

// SetNotReadyConditionWaiting sets the WaitingFor condition on an OpenStack resource. It returns the condition that was set.
func SetNotReadyConditionWaiting(openStackResource, patch openstackv1.OpenStackResourceCommonStatus, objects []Dependency) (bool, *metav1.Condition) {
	return SetCondition(openStackResource, patch, NotReadyWaiting(objects))
}

func NotReadyTransientError(errorMessage string) metav1.Condition {
	return metav1.Condition{
		Type:               string(openstackv1.OpenStackConditionReady),
		Status:             metav1.ConditionFalse,
		Reason:             "TransientError",
		Message:            errorMessage,
		LastTransitionTime: metav1.Now(),
	}
}

func SetNotReadyConditionTransientError(openStackResource, patch openstackv1.OpenStackResourceCommonStatus, errorMessage string) (bool, *metav1.Condition) {
	return SetCondition(openStackResource, patch, NotReadyTransientError(errorMessage))
}

func NotReadyError(errorMessage string) metav1.Condition {
	return metav1.Condition{
		Type:               string(openstackv1.OpenStackConditionReady),
		Status:             metav1.ConditionFalse,
		Reason:             "Error",
		Message:            errorMessage,
		LastTransitionTime: metav1.Now(),
	}
}

func SetNotReadyConditionError(openStackResource, patch openstackv1.OpenStackResourceCommonStatus, errorMessage string) (bool, *metav1.Condition) {
	return SetCondition(openStackResource, patch, NotReadyTransientError(errorMessage))
}

func NotReadyPending() metav1.Condition {
	return metav1.Condition{
		Type:               string(openstackv1.OpenStackConditionReady),
		Status:             metav1.ConditionFalse,
		Reason:             "Pending",
		Message:            "Pending",
		LastTransitionTime: metav1.Now(),
	}
}

func SetNotReadyConditionPending(openStackResource, patch openstackv1.OpenStackResourceCommonStatus) (bool, *metav1.Condition) {
	return SetCondition(openStackResource, patch, NotReadyPending())
}

func ReadyCondition(ready bool) metav1.Condition {
	var status metav1.ConditionStatus
	var reason string
	if ready {
		status = metav1.ConditionTrue
		reason = OpenStackConditionReasonReady
	} else {
		status = metav1.ConditionFalse
		reason = OpenStackConditionReadonNotReady
	}

	return metav1.Condition{
		Type:               string(openstackv1.OpenStackConditionReady),
		Status:             status,
		Reason:             reason,
		Message:            "Ready",
		LastTransitionTime: metav1.Now(),
	}
}

// SetReadyCondition sets the Ready condition on an OpenStack resource. It returns the condition that was set.
func SetReadyCondition(openStackResource, patch openstackv1.OpenStackResourceCommonStatus) (bool, *metav1.Condition) {
	return SetCondition(openStackResource, patch, ReadyCondition(true))
}

func ErrorCondition(errorReason, errorMessage string) metav1.Condition {
	if errorReason == "" {
		errorReason = OpenStackConditionReasonNoError
	}

	status := metav1.ConditionFalse
	if errorReason != OpenStackConditionReasonNoError {
		status = metav1.ConditionTrue
	}

	return metav1.Condition{
		Type:               string(openstackv1.OpenStackConditionError),
		Status:             status,
		Reason:             errorReason,
		Message:            errorMessage,
		LastTransitionTime: metav1.Now(),
	}
}

// SetErrorCondition sets the Error condition on an OpenStack resource. It returns the condition that was set.
func SetErrorCondition(openStackResource, patch openstackv1.OpenStackResourceCommonStatus, errorReason, errorMessage string) (bool, *metav1.Condition) {
	SetNotReadyConditionError(openStackResource, patch, errorMessage)
	return SetCondition(openStackResource, patch, ErrorCondition(errorReason, errorMessage))
}

// InitialiseRequiredConditions initialises an empty set of required conditions in an OpenStack resource.
func InitialiseRequiredConditions(openStackResource, patch openstackv1.OpenStackResourceCommonStatus) {
	for _, condition := range []metav1.Condition{
		NotReadyPending(),
		ErrorCondition("", ""),
	} {
		SetCondition(openStackResource, patch, condition)
	}
}

func ConditionMatches(a, b *metav1.Condition) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Status == b.Status && a.Reason == b.Reason && a.Message == b.Message
}

// EmitEventForCondition emits an event for the given condition on the OpenStack resource.
func EmitEventForCondition(recorder record.EventRecorder, openStackResource runtime.Object, eventType string, condition *metav1.Condition) {
	recorder.Event(openStackResource, eventType, condition.Reason, condition.Message)
}
