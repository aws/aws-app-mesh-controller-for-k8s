package controller

import (
	"context"
	"fmt"
	"time"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	meshclientset "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned"
	meshscheme "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned/scheme"
	meshinformers "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/informers/externalversions/appmesh/v1beta1"
	meshlisters "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/listers/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const (
	//setting default threadiness to number of go-routines
	DefaultThreadiness                  = 5
	controllerAgentName                 = "app-mesh-controller"
	meshDeletionFinalizerName           = "meshDeletion.finalizers.appmesh.k8s.aws"
	virtualNodeDeletionFinalizerName    = "virtualNodeDeletion.finalizers.appmesh.k8s.aws"
	virtualServiceDeletionFinalizerName = "virtualServiceDeletion.finalizers.appmesh.k8s.aws"
)

type Controller struct {
	name  string
	cloud aws.CloudAPI
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// meshclientset is a clientset for our own API group
	meshclientset meshclientset.Interface

	podsLister corev1listers.PodLister
	podsSynced cache.InformerSynced

	meshLister           meshlisters.MeshLister
	meshIndex            cache.Indexer
	meshSynced           cache.InformerSynced
	virtualNodeLister    meshlisters.VirtualNodeLister
	virtualNodeIndex     cache.Indexer
	virtualNodeSynced    cache.InformerSynced
	virtualServiceLister meshlisters.VirtualServiceLister
	virtualServiceIndex  cache.Indexer
	virtualServiceSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	mq workqueue.RateLimitingInterface
	nq workqueue.RateLimitingInterface
	sq workqueue.RateLimitingInterface
	pq workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// stats records mesh Prometheus metrics
	stats *metrics.Recorder
}

func NewController(
	cloud aws.CloudAPI,
	kubeclientset kubernetes.Interface,
	meshclientset meshclientset.Interface,
	podInformer coreinformers.PodInformer,
	meshInformer meshinformers.MeshInformer,
	virtualNodeInformer meshinformers.VirtualNodeInformer,
	virtualServiceInformer meshinformers.VirtualServiceInformer,
	stats *metrics.Recorder) (*Controller, error) {

	utilruntime.Must(meshscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		name:                 controllerAgentName,
		cloud:                cloud,
		kubeclientset:        kubeclientset,
		meshclientset:        meshclientset,
		podsLister:           podInformer.Lister(),
		podsSynced:           podInformer.Informer().HasSynced,
		meshLister:           meshInformer.Lister(),
		meshSynced:           meshInformer.Informer().HasSynced,
		virtualNodeLister:    virtualNodeInformer.Lister(),
		virtualNodeSynced:    virtualNodeInformer.Informer().HasSynced,
		virtualServiceLister: virtualServiceInformer.Lister(),
		virtualServiceSynced: virtualServiceInformer.Informer().HasSynced,
		mq:                   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		nq:                   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		sq:                   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		pq:                   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		recorder:             recorder,
		stats:                stats,
	}

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.podAdded,
		UpdateFunc: controller.podUpdated,
		DeleteFunc: controller.podDeleted,
	})

	meshInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.meshAdded,
		UpdateFunc: controller.meshUpdated,
		DeleteFunc: controller.meshDeleted,
	})

	virtualNodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.virtualNodeAdded,
		UpdateFunc: controller.virtualNodeUpdated,
		DeleteFunc: controller.virtualNodeDeleted,
	})

	if err := virtualNodeInformer.Informer().GetIndexer().AddIndexers(cache.Indexers{
		"meshName": indexVNodesByMeshName,
	}); err != nil {
		return nil, fmt.Errorf("failed to add meshName index: %s", err)
	}

	controller.virtualNodeIndex = virtualNodeInformer.Informer().GetIndexer()

	virtualServiceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.virtualServiceAdded,
		UpdateFunc: controller.virtualServiceUpdated,
		DeleteFunc: controller.virtualServiceDeleted,
	})

	if err := virtualServiceInformer.Informer().GetIndexer().AddIndexers(cache.Indexers{
		"meshName": indexVServicesByMeshName,
	}); err != nil {
		return nil, fmt.Errorf("failed to add meshName index: %s", err)
	}

	controller.virtualServiceIndex = virtualServiceInformer.Informer().GetIndexer()

	controller.meshIndex = meshInformer.Informer().GetIndexer()

	return controller, nil
}

func indexVNodesByMeshName(obj interface{}) ([]string, error) {
	node, ok := obj.(*appmeshv1beta1.VirtualNode)
	if !ok {
		return []string{}, nil
	}
	// MeshName must be set
	if len(node.Spec.MeshName) == 0 {
		return []string{}, nil
	}
	return []string{node.Spec.MeshName}, nil
}

func indexVServicesByMeshName(obj interface{}) ([]string, error) {
	node, ok := obj.(*appmeshv1beta1.VirtualService)
	if !ok {
		return []string{}, nil
	}
	// MeshName must be set
	if len(node.Spec.MeshName) == 0 {
		return []string{}, nil
	}
	return []string{node.Spec.MeshName}, nil
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) error {
	klog.Info("Starting controller")

	defer runtime.HandleCrash()
	defer c.mq.ShutDown()
	defer c.nq.ShutDown()
	defer c.sq.ShutDown()
	defer c.pq.ShutDown()

	// Start the informer factories to begin populating the informer caches
	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.podsSynced, c.meshSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch workers to process Mesh resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.meshWorker, time.Second, stopCh)
		go wait.Until(c.vNodeWorker, time.Second, stopCh)
		go wait.Until(c.vServiceWorker, time.Second, stopCh)
		go wait.Until(c.podWorker, time.Second, stopCh)
		go wait.Until(c.cloudmapReconciler, 1*time.Minute, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// podAdded adds the pods endpoint to matching CloudMap Services.
func (c *Controller) podAdded(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.V(4).Infof("Error getting key for obj")
	}
	c.pq.Add(key)
}

func (c *Controller) podUpdated(old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err != nil {
		klog.V(4).Infof("Error getting key for obj")
	}
	c.pq.Add(key)
}

// podDeleted removes the endpoint from matching CloudMap services
func (c *Controller) podDeleted(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.V(4).Infof("Error getting key for obj")
	}
	c.pq.Add(key)
}

// getMeshServicesForPod finds Mesh Services with selectors that match the Pod's labels
func (c *Controller) getMeshServicesForPod(pod *corev1.Pod) ([]*appmeshv1beta1.VirtualService, error) {
	return nil, nil
}

func (c *Controller) meshAdded(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	klog.V(4).Infof("Mesh Added: %s", key)
	if err == nil {
		c.mq.Add(key)
	} else {
		utilruntime.HandleError(fmt.Errorf("mesh key error for %s: %s", key, err))
		return
	}

	mesh := obj.(*appmeshv1beta1.Mesh)
	meshName := mesh.Name

	// If a mesh is created, process all objects with the meshName.
	c.enqueueVNodesForMesh(meshName)
	c.enqueueVServicesForMesh(meshName)
}

func (c *Controller) enqueueVNodesForMesh(name string) {
	if objects, err := c.virtualNodeIndex.ByIndex("meshName", name); err != nil {
		utilruntime.HandleError(fmt.Errorf("meshName index error for %s: %s", name, err))
		return
	} else {
		for _, obj := range objects {
			vnode, ok := obj.(*appmeshv1beta1.VirtualNode)
			if !ok {
				continue
			}

			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.nq.Add(key)
			} else {
				continue
			}
			klog.Infof("Processed virtual node %s due to new mesh.", vnode.Name)
		}
	}
}

func (c *Controller) enqueueVServicesForMesh(name string) {
	if objects, err := c.virtualServiceIndex.ByIndex("meshName", name); err != nil {
		utilruntime.HandleError(fmt.Errorf("meshName index error for %s: %s", name, err))
		return
	} else {
		for _, obj := range objects {
			vservice, ok := obj.(*appmeshv1beta1.VirtualService)
			if !ok {
				continue
			}

			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.nq.Add(key)
			} else {
				continue
			}
			klog.Infof("Processed virtual service %s due to new mesh.", vservice.Name)
		}
	}
}

func (c *Controller) meshUpdated(old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		klog.V(4).Infof("Mesh Updated: %s", key)
		c.mq.Add(key)
	}
}

func (c *Controller) meshDeleted(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		klog.V(4).Infof("Mesh Deleted: %s", key)
		c.mq.Add(key)
	}
}

func (c *Controller) virtualNodeAdded(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		klog.V(4).Infof("Virtual Node Added: %s", key)
		c.nq.Add(key)
	}
}

func (c *Controller) virtualNodeUpdated(old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		klog.V(4).Infof("Virtual Node Updated: %s", key)
		c.nq.Add(key)
	}
}

func (c *Controller) virtualNodeDeleted(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		klog.V(4).Infof("Virtual Node Deleted: %s", key)
		c.nq.Add(key)
	}
}

func (c *Controller) virtualServiceAdded(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		klog.V(4).Infof("Virtual Service Added: %s", key)
		c.sq.Add(key)
	}
}

func (c *Controller) virtualServiceUpdated(old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		klog.V(4).Infof("Virtual Service Updated: %s", key)
		c.sq.Add(key)
	}
}

func (c *Controller) virtualServiceDeleted(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		klog.V(4).Infof("Virtual Service Deleted: %s", key)
		c.sq.Add(key)
	}
}

func (c *Controller) meshWorker() {
	for c.processNext(c.mq, c.handleMesh) {
	}
}

func (c *Controller) vNodeWorker() {
	for c.processNext(c.nq, c.handleVNode) {
	}
}

func (c *Controller) vServiceWorker() {
	for c.processNext(c.sq, c.handleVService) {
	}
}

func (c *Controller) podWorker() {
	for c.processNext(c.pq, c.handlePod) {
	}
}

func (c *Controller) cloudmapReconciler() {
	ctx := context.Background()
	c.reconcileServices(ctx)
	c.reconcileInstances(ctx)
}

// processNext will read a single work item off the queue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNext(queue workqueue.RateLimitingInterface, syncHandler func(key string) error) bool {
	return processNextWorkItem(queue, syncHandler)
}

func processNextWorkItem(queue workqueue.RateLimitingInterface, syncHandler func(key string) error) bool {
	obj, shutdown := queue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off period.
		defer queue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			queue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// resource to be synced.
		if err := syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		queue.Forget(obj)
		klog.V(4).Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}
