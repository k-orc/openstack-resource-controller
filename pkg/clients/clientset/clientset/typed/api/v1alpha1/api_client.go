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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	http "net/http"

	apiv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	scheme "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/clientset/clientset/scheme"
	rest "k8s.io/client-go/rest"
)

type OpenstackV1alpha1Interface interface {
	RESTClient() rest.Interface
	FlavorsGetter
	FloatingIPsGetter
	ImagesGetter
	NetworksGetter
	PortsGetter
	ProjectsGetter
	RoutersGetter
	RouterInterfacesGetter
	SecurityGroupsGetter
	ServersGetter
	ServerGroupsGetter
	SubnetsGetter
}

// OpenstackV1alpha1Client is used to interact with features provided by the openstack.k-orc.cloud group.
type OpenstackV1alpha1Client struct {
	restClient rest.Interface
}

func (c *OpenstackV1alpha1Client) Flavors(namespace string) FlavorInterface {
	return newFlavors(c, namespace)
}

func (c *OpenstackV1alpha1Client) FloatingIPs(namespace string) FloatingIPInterface {
	return newFloatingIPs(c, namespace)
}

func (c *OpenstackV1alpha1Client) Images(namespace string) ImageInterface {
	return newImages(c, namespace)
}

func (c *OpenstackV1alpha1Client) Networks(namespace string) NetworkInterface {
	return newNetworks(c, namespace)
}

func (c *OpenstackV1alpha1Client) Ports(namespace string) PortInterface {
	return newPorts(c, namespace)
}

func (c *OpenstackV1alpha1Client) Projects(namespace string) ProjectInterface {
	return newProjects(c, namespace)
}

func (c *OpenstackV1alpha1Client) Routers(namespace string) RouterInterface {
	return newRouters(c, namespace)
}

func (c *OpenstackV1alpha1Client) RouterInterfaces(namespace string) RouterInterfaceInterface {
	return newRouterInterfaces(c, namespace)
}

func (c *OpenstackV1alpha1Client) SecurityGroups(namespace string) SecurityGroupInterface {
	return newSecurityGroups(c, namespace)
}

func (c *OpenstackV1alpha1Client) Servers(namespace string) ServerInterface {
	return newServers(c, namespace)
}

func (c *OpenstackV1alpha1Client) ServerGroups(namespace string) ServerGroupInterface {
	return newServerGroups(c, namespace)
}

func (c *OpenstackV1alpha1Client) Subnets(namespace string) SubnetInterface {
	return newSubnets(c, namespace)
}

// NewForConfig creates a new OpenstackV1alpha1Client for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*OpenstackV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

// NewForConfigAndClient creates a new OpenstackV1alpha1Client for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*OpenstackV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &OpenstackV1alpha1Client{client}, nil
}

// NewForConfigOrDie creates a new OpenstackV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *OpenstackV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new OpenstackV1alpha1Client for the given RESTClient.
func New(c rest.Interface) *OpenstackV1alpha1Client {
	return &OpenstackV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := apiv1alpha1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = rest.CodecFactoryForGeneratedClient(scheme.Scheme, scheme.Codecs).WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *OpenstackV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
