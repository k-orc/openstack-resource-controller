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

package generic

import (
	"fmt"
	"time"
)

type WaitingOnEvent interface {
	Message() string
	Requeue() time.Duration
}

type waitingOnType int

const (
	WaitingOnCreation waitingOnType = iota
	WaitingOnReady
	WaitingOnDeletion
)

type waitingOnORC struct {
	kind      string
	name      string
	waitingOn waitingOnType
}

var _ WaitingOnEvent = waitingOnORC{}

func (e waitingOnORC) Message() string {
	var outcome string
	switch e.waitingOn {
	case WaitingOnCreation:
		outcome = "created"
	case WaitingOnReady:
		outcome = "ready"
	case WaitingOnDeletion:
		outcome = "deleted"
	}
	return fmt.Sprintf("Waiting for %s/%s to be %s", e.kind, e.name, outcome)
}

func newWaitingOnORC(kind, name string, event waitingOnType) WaitingOnEvent {
	return waitingOnORC{
		kind:      kind,
		name:      name,
		waitingOn: event,
	}
}

func WaitingOnORCExist(kind, name string) WaitingOnEvent {
	return newWaitingOnORC(kind, name, WaitingOnCreation)
}

func WaitingOnORCReady(kind, name string) WaitingOnEvent {
	return newWaitingOnORC(kind, name, WaitingOnReady)
}

func WaitingOnORCDeleted(kind, name string) WaitingOnEvent {
	return newWaitingOnORC(kind, name, WaitingOnDeletion)
}

func (e waitingOnORC) Requeue() time.Duration {
	return 0
}

type waitingOnFinalizer struct {
	finalizer string
}

func (e waitingOnFinalizer) Message() string {
	return fmt.Sprintf("Waiting for finalizer %s to be removed", e.finalizer)
}

func (e waitingOnFinalizer) Requeue() time.Duration {
	return 0
}

func WaitingOnFinalizer(finalizer string) WaitingOnEvent {
	return waitingOnFinalizer{finalizer: finalizer}
}

type waitingOnOpenStack struct {
	waitingOn     waitingOnType
	pollingPeriod time.Duration
}

var _ WaitingOnEvent = waitingOnOpenStack{}

func newWaitingOnOpenStack(event waitingOnType, pollingPeriod time.Duration) WaitingOnEvent {
	return waitingOnOpenStack{
		waitingOn:     event,
		pollingPeriod: pollingPeriod,
	}
}

func WaitingOnOpenStackCreate(pollingPeriod time.Duration) WaitingOnEvent {
	return newWaitingOnOpenStack(WaitingOnCreation, pollingPeriod)
}

func WaitingOnOpenStackReady(pollingPeriod time.Duration) WaitingOnEvent {
	return newWaitingOnOpenStack(WaitingOnReady, pollingPeriod)
}

func WaitingOnOpenStackDeleted(pollingPeriod time.Duration) WaitingOnEvent {
	return newWaitingOnOpenStack(WaitingOnDeletion, pollingPeriod)
}

func (e waitingOnOpenStack) Message() string {
	var outcome string
	switch e.waitingOn {
	case WaitingOnCreation:
		outcome = "be created externally"
	case WaitingOnReady:
		outcome = "be ready"
	case WaitingOnDeletion:
		outcome = "be deleted"
	}
	return fmt.Sprintf("Waiting for OpenStack resource to %s", outcome)
}

func (e waitingOnOpenStack) Requeue() time.Duration {
	return e.pollingPeriod
}

func MaxRequeue(evts []WaitingOnEvent) time.Duration {
	var ret time.Duration
	for _, evt := range evts {
		if evt.Requeue() > ret {
			ret = evt.Requeue()
		}
	}
	return ret
}
