########################################################################################################################

package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gophercloud/gophercloud"
    "github.com/gophercloud/gophercloud/openstack"
    "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/addressscopes"
    networkingv1 "github.com/FaizFarooqMoazam/openstack-resource-controller/api/v1"
    clientset "github.com/FaizFarooqMoazam/openstack-resource-controller/generated/clientset/versioned"
    informers "github.com/FaizFarooqMoazam/openstack-resource-controller/generated/informers/externalversions/networking/v1"
    listers "github.com/FaizFarooqMoazam/openstack-resource-controller/generated/listers/networking/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    utilruntime "k8s.io/apimachinery/pkg/util/runtime"
    "k8s.io/apimachinery/pkg/util/wait"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/cache"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/workqueue"
    "k8s.io/klog/v2"
)

type Controller struct {
    neutronClient *gophercloud.ServiceClient
    kubeClient    kubernetes.Interface
    crdClient     clientset.Interface

    informer  cache.SharedIndexInformer
    lister    listers.AddressScopeLister
    workqueue workqueue.RateLimitingInterface
}

func NewController(neutronClient *gophercloud.ServiceClient,
    kubeClient kubernetes.Interface,
    crdClient clientset.Interface,
    informer informers.AddressScopeInformer) *Controller {

    c := &Controller{
        neutronClient: neutronClient,
        kubeClient:    kubeClient,
        crdClient:     crdClient,
        informer:      informer.Informer(),
        lister:        informer.Lister(),
        workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "addressscopes"),
    }

    informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc:    c.enqueueAddressScope,
        UpdateFunc: func(oldObj, newObj interface{}) { c.enqueueAddressScope(newObj) },
        DeleteFunc: c.enqueueAddressScope,
    })

    return c
}

func (c *Controller) enqueueAddressScope(obj interface{}) {
    key, err := cache.MetaNamespaceKeyFunc(obj)
    if err != nil {
        utilruntime.HandleError(fmt.Errorf("error getting key for object: %v", err))
        return
    }
    c.workqueue.Add(key)
    klog.Infof("Enqueued AddressScope: %s", key)
}

func (c *Controller) runWorker(ctx context.Context) {
    for c.processNextWorkItem(ctx) {
    }
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
    obj, shutdown := c.workqueue.Get()
    if shutdown {
        return false
    }
    defer c.workqueue.Done(obj)

    key, ok := obj.(string)
    if !ok {
        c.workqueue.Forget(obj)
        utilruntime.HandleError(fmt.Errorf("expected string in queue but got %#v", obj))
        return true
    }

    if err := c.reconcile(ctx, key); err != nil {
        utilruntime.HandleError(fmt.Errorf("error syncing AddressScope %q: %v", key, err))
        c.workqueue.AddRateLimited(key)
        return true
    }
    c.workqueue.Forget(obj)
    return true
}

func (c *Controller) reconcile(ctx context.Context, key string) error {
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        return err
    }
    as, err := c.lister.AddressScopes(namespace).Get(name)
    if errors.IsNotFound(err) {
        // Resource deleted, attempt to delete from Neutron
        return c.deleteNeutronAddressScope(ctx, key)
    } else if err != nil {
        return err
    }

    // Create or update Neutron Address Scope
    return c.syncNeutronAddressScope(ctx, as)
}

func (c *Controller) syncNeutronAddressScope(ctx context.Context, as *networkingv1.AddressScope) error {
    asName := as.Spec.Name
    ipVersion := as.Spec.IPVersion
    description := as.Spec.Description
    shared := gophercloud.Disabled
    if as.Spec.Shared {
        shared = gophercloud.Enabled
    }

    // Check if exists
    existingPages, err := addressscopes.List(c.neutronClient, addressscopes.ListOpts{
        Name: asName,
    }).AllPages()
    if err != nil {
        return fmt.Errorf("error listing neutron address scopes: %v", err)
    }
    existingScopes, err := addressscopes.ExtractAddressScopes(existingPages)
    if err != nil {
        return fmt.Errorf("error extracting neutron address scopes: %v", err)
    }

    if len(existingScopes) == 0 {
        // Create new
        opts := addressscopes.CreateOpts{
            Name:        asName,
            IPVersion:   ipVersion,
            Description: description,
            Shared:      shared,
        }
        newAS, err := addressscopes.Create(c.neutronClient, opts).Extract()
        if err != nil {
            return fmt.Errorf("error creating neutron address scope: %v", err)
        }
        klog.Infof("Created Neutron AddressScope %s (%s)", newAS.Name, newAS.ID)

        // Update status with Neutron ID and state
        as.Status.NeutronID = newAS.ID
        as.Status.State = "Created"
        _, err = c.crdClient.NetworkingV1().AddressScopes(as.Namespace).UpdateStatus(ctx, as, metav1.UpdateOptions{})
        if err != nil {
            klog.Errorf("Failed to update AddressScope status: %v", err)
        }
        return nil
    }

    // Update existing
    existing := existingScopes[0]
    opts := addressscopes.UpdateOpts{
        Description: &description,
        Shared:      &shared,
    }
    _, err = addressscopes.Update(c.neutronClient, existing.ID, opts).Extract()
    if err != nil {
        return fmt.Errorf("error updating neutron address scope: %v", err)
    }
    klog.Infof("Updated Neutron AddressScope %s (%s)", existing.Name, existing.ID)

    // Update status with Neutron ID and state
    as.Status.NeutronID = existing.ID
    as.Status.State = "Updated"
    _, err = c.crdClient.NetworkingV1().AddressScopes(as.Namespace).UpdateStatus(ctx, as, metav1.UpdateOptions{})
    if err != nil {
        klog.Errorf("Failed to update AddressScope status: %v", err)
    }
    return nil
}

func (c *Controller) deleteNeutronAddressScope(ctx context.Context, key string) error {
    // Try to find Neutron ID from cache or another store, if this doesn't work, simply ignore
    // For now, I try to log deletion attempts and just continue

    klog.Infof("Resource %s deleted from Kubernetes, implement Neutron deletion here if needed.", key)
    return nil
}

func main() {
    // Setup kubeconfig and clients
    var config *rest.Config
    var err error
    if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
        config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
    } else {
        config, err = rest.InClusterConfig()
    }
    if err != nil {
        panic(fmt.Errorf("failed to build kubeconfig: %v", err))
    }

    kubeClient, err := kubernetes.NewForConfig(config)
    if err != nil {
        panic(err)
    }
    crdClient, err := clientset.NewForConfig(config)
    if err != nil {
        panic(err)
    }

    authOpts := gophercloud.AuthOptions{
        IdentityEndpoint: "http://keystone.openstack.svc:5000/v3",
        Username:         "admin",
        Password:         "password",
        DomainName:       "Default",
        TenantName:       "admin",
    }
    provider, err := openstack.AuthenticatedClient(authOpts)
    if err != nil {
        panic(err)
    }
    neutronClient, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{})
    if err != nil {
        panic(err)
    }

    informerFactory := informers.NewSharedInformerFactory(crdClient, time.Minute*10)
    addressScopeInformer := informerFactory.AddressScopes()

    controller := NewController(neutronClient, kubeClient, crdClient, addressScopeInformer)

    stopCh := make(chan struct{})
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go func() {
        sigs := make(chan os.Signal, 1)
        signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
        <-sigs
        klog.Info("Shutdown signal received, exiting...")
        cancel()
        close(stopCh)
    }()

    informerFactory.Start(stopCh)
    if !cache.WaitForCacheSync(stopCh, controller.informer.HasSynced) {
        klog.Fatalf("Failed to sync caches")
    }

    klog.Info("Starting worker...")
    wait.UntilWithContext(ctx, controller.runWorker, time.Second)
}