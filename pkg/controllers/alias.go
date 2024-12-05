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

package controllers

// This file provides a minimal exported interface to non-exported controllers.

import (
	"github.com/k-orc/openstack-resource-controller/internal/controllers/flavor"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/image"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/network"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/port"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/router"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/routerinterface"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/securitygroup"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/server"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/subnet"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
)

var ImageController = image.New
var NetworkController = network.New
var SubnetController = subnet.New
var RouterController = router.New
var RouterInterfaceController = routerinterface.New
var PortController = port.New
var FlavorController = flavor.New
var SecurityGroupController = securitygroup.New
var ServerController = server.New

var NewScopeFactory = scope.NewFactory
