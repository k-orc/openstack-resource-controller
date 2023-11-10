package cloud

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openstackv1 "github.com/gophercloud/gopherkube/api/v1alpha1"
)

func NewClient(ctx context.Context, k8sClient client.Client, openStackCloud *openstackv1.OpenStackCloud, service string) (*gophercloud.ServiceClient, error) {
	if source := openStackCloud.Spec.Credentials.Source; source != "secret" {
		return nil, fmt.Errorf("unknown credentials source %q", source)
	}
	secret := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Namespace: openStackCloud.GetNamespace(),
		Name:      openStackCloud.Spec.Credentials.SecretRef.Name,
	}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("cloud secret %q not found: %w", openStackCloud.Spec.Credentials.SecretRef.Name, err)
		}
		return nil, fmt.Errorf("error retrieving cloud secret %q: %w", openStackCloud.Spec.Credentials.SecretRef.Name, err)
	}

	cloudBytes, ok := secret.Data[openStackCloud.Spec.Credentials.SecretRef.Key]
	if !ok {
		return nil, fmt.Errorf("key %q not found in cloud secret %q", openStackCloud.Spec.Credentials.SecretRef.Key, openStackCloud.Spec.Credentials.SecretRef.Name)
	}

	var clouds clientconfig.Clouds

	if err := yaml.Unmarshal(cloudBytes, &clouds); err != nil {
		return nil, fmt.Errorf("unmarshaling clouds.yaml: %w", err)
	}

	cloud, ok := clouds.Clouds[openStackCloud.Spec.Cloud]
	if !ok {
		return nil, fmt.Errorf("cloud %q not found in clouds.yaml", openStackCloud.Spec.Cloud)
	}

	providerClient, err := openstack.AuthenticatedClient(gophercloud.AuthOptions{
		IdentityEndpoint:            cloud.AuthInfo.AuthURL,
		Username:                    cloud.AuthInfo.Username,
		UserID:                      cloud.AuthInfo.UserID,
		Password:                    cloud.AuthInfo.Password,
		DomainID:                    cloud.AuthInfo.DomainID,
		DomainName:                  cloud.AuthInfo.UserDomainName,
		TenantID:                    cloud.AuthInfo.ProjectID,
		TenantName:                  cloud.AuthInfo.ProjectName,
		AllowReauth:                 cloud.AuthInfo.AllowReauth,
		TokenID:                     cloud.AuthInfo.Token,
		ApplicationCredentialID:     cloud.AuthInfo.ApplicationCredentialID,
		ApplicationCredentialName:   cloud.AuthInfo.ApplicationCredentialName,
		ApplicationCredentialSecret: cloud.AuthInfo.ApplicationCredentialSecret,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating an OpenStack provider client: %w", err)
	}

	eo := gophercloud.EndpointOpts{
		Region:       cloud.RegionName,
		Availability: gophercloud.AvailabilityPublic,
	}

	switch service {
	case "compute":
		return openstack.NewComputeV2(providerClient, eo)
	case "image":
		return openstack.NewImageServiceV2(providerClient, eo)
	case "network":
		return openstack.NewNetworkV2(providerClient, eo)
	default:
		return nil, fmt.Errorf("unable to create a service client for %s", service)
	}
}
