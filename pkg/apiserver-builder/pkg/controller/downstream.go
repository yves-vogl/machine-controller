package controller

import (
	"time"

	"github.com/golang/glog"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Copied from github.com/kubernetes-incubator/apiserver-builder/pkg/controller/informers.go
type SharedInformersDefaults struct {
	KubernetesFactory   informers.SharedInformerFactory
	KubernetesClientSet *kubernetes.Clientset

	// Extensions allows a controller-manager to define new data structures
	// shared by all of its controllers.
	// Set this by overriding the InitExtensions function on the generated *SharedInformers
	// type under the consuming projects pkg/controller/sharedinformers package
	// by in a new informers.go file
	Extensions interface{}

	WorkerQueues map[string]*QueueWorker
}

// Copied from github.com/kubernetes-incubator/apiserver-builder/pkg/controller/controller.go
type QueueWorker struct {
	Queue      workqueue.RateLimitingInterface
	MaxRetries int
	Name       string
	Reconcile  func(key string) error
}

// Copied from github.com/kubernetes-incubator/apiserver-builder/pkg/controller/controller.go
type QueueingEventHandler struct {
	Queue         workqueue.RateLimitingInterface
	ObjToKey      func(obj interface{}) (string, error)
	EnqueueDelete bool
}

// Copied from github.com/kubernetes-incubator/apiserver-builder/pkg/controller/controller.go
func (c *QueueingEventHandler) enqueue(obj interface{}) {
	fn := c.ObjToKey
	if c.ObjToKey == nil {
		fn = cache.DeletionHandlingMetaNamespaceKeyFunc
	}
	key, err := fn(obj)
	if err != nil {
		glog.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}
	c.Queue.Add(key)
}

// Copied from github.com/kubernetes-incubator/apiserver-builder/pkg/controller/controller.go
func (c *QueueingEventHandler) OnAdd(obj interface{}) {
	glog.V(6).Infof("Add event for %+v\n", obj)
	c.enqueue(obj)
}

// Copied from github.com/kubernetes-incubator/apiserver-builder/pkg/controller/controller.go
func (c *QueueingEventHandler) OnUpdate(oldObj, newObj interface{}) {
	glog.V(6).Infof("Update event for %+v\n", newObj)
	c.enqueue(newObj)
}

// Copied from github.com/kubernetes-incubator/apiserver-builder/pkg/controller/controller.go
func (c *QueueingEventHandler) OnDelete(obj interface{}) {
	glog.V(6).Infof("Delete event for %+v\n", obj)
	if c.EnqueueDelete {
		c.enqueue(obj)
	}
}

// Copied from github.com/kubernetes-incubator/apiserver-builder/pkg/controller/informers.go
func (si *SharedInformersDefaults) InitKubernetesInformers(config *rest.Config) {
	si.KubernetesClientSet = kubernetes.NewForConfigOrDie(config)
	si.KubernetesFactory = informers.NewSharedInformerFactory(si.KubernetesClientSet, 10*time.Minute)
}

// Copied from github.com/kubernetes-incubator/apiserver-builder/pkg/controller/informers.go
func (*SharedInformersDefaults) Init() {}

// Copied from github.com/kubernetes-incubator/apiserver-builder/pkg/controller/informers.go
func (c *SharedInformersDefaults) Watch(
	name string, i cache.SharedIndexInformer,
	f func(interface{}) (string, error), r func(string) error) {
	q := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name)

	queue := &QueueWorker{q, 10, name, r}
	if c.WorkerQueues == nil {
		c.WorkerQueues = map[string]*QueueWorker{}
	}
	c.WorkerQueues[name] = queue
	i.AddEventHandler(&QueueingEventHandler{q, f, true})
}
