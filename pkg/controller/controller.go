package controller

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/kahootali/Kubernetes-Controller/pkg/config"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// AllNamespaces as our controller will be looking for events in all namespaces
const (
	AllNamespaces = ""
)

// Event which is used to send additional information than just the key, can have other entiites
type Event struct {
	key       string
	eventType string
	podName   string
}

// Controller for checking items
type Controller struct {
	clientset *kubernetes.Clientset
	indexer   cache.Indexer
	queue     workqueue.RateLimitingInterface
	informer  cache.Controller
	config    config.Configuration
}

// NewController A Constructor for the Controller to initialize the controller
func NewController(clientset *kubernetes.Clientset, conf config.Configuration) *Controller {

	controller := &Controller{
		clientset: clientset,
		config:    conf,
	}
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	listWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", AllNamespaces, fields.Everything())

	indexer, informer := cache.NewIndexerInformer(listWatcher, &v1.Pod{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.Add,    //function that is called when the object is created
		UpdateFunc: controller.Update, //function that is called when the object is updated
		DeleteFunc: controller.Delete, //function that is called when the object is deleted
	}, cache.Indexers{})

	controller.indexer = indexer
	controller.informer = informer
	controller.queue = queue
	return controller

}

//Add function to add a 'create' event to the queue
func (c *Controller) Add(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	var event Event

	if err == nil {
		event.key = key
		event.eventType = "create"
		event.podName = obj.(*v1.Pod).Name
		c.queue.Add(event)
	}
}

//Update function to add an 'update' event to the queue
func (c *Controller) Update(oldObj interface{}, newObj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	var event Event

	if err == nil {
		event.key = key
		event.eventType = "update"
		event.podName = oldObj.(*v1.Pod).Name
		c.queue.Add(event)
	}
}

//Delete function to add a 'delete' event to the queue
func (c *Controller) Delete(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	var event Event

	if err == nil {
		event.key = key
		event.eventType = "delete"
		event.podName = obj.(*v1.Pod).Name
		c.queue.Add(event)
	}
}

//Run function for controller which handles the queue
func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	event, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two ingresses with the same key are never processed in
	// parallel.
	defer c.queue.Done(event)

	// Invoke the method containing the business logic
	err := c.takeAction(event.(Event))
	// Handle the error if something went wrong during the execution of the business logic
	c.handleErr(err, event)
	return true
}

//takeAction, the main function which will be handling the controller business logic
func (c *Controller) takeAction(event Event) error {

	obj, _, err := c.indexer.GetByKey(event.key)
	if err != nil {
		log.Printf("Fetching object with key %s from store failed with %v", event.key, err)
	}

	// process events based on its type
	switch event.eventType {
	//Printing Pod Name and its Containers from all namespaces but we can do anything in these functions
	case "create":
		objectCreated(obj)

	case "update":
		objectUpdated(obj)

	case "delete":
		objectDeleted(event.podName) //Incase of deleted, the obj object is nil
	}

	return nil
}
func objectCreated(obj interface{}) {
	// Currently printing all pods but we can restrict using any of the data in yaml file
	// e.g If want to check on APIVersion uncomment this.
	// if obj.(*v1.Pod).APIVersion == "samplecontroller.k8s.io/v1alpha1" {

	fmt.Println("\nA Pod with name " + obj.(*v1.Pod).Name + " has been created.")
	printContainersOfPod(obj)

	// }
}
func objectUpdated(obj interface{}) {
	fmt.Println("\nA Pod with name " + obj.(*v1.Pod).Name + " has been updated.")
	printContainersOfPod(obj)

}
func objectDeleted(podName string) {
	fmt.Println("\nA Pod with name " + podName + " has been deleted.")
}
func printContainersOfPod(obj interface{}) {

	fmt.Println("Containers in Pod " + obj.(*v1.Pod).Name + " : ")
	for index, container := range obj.(*v1.Pod).Spec.Containers {
		fmt.Println(strconv.Itoa(index+1) + ".  " + container.Name)
	}
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		log.Printf("Error syncing ingress %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	log.Printf("Dropping ingress %q out of the queue: %v", key, err)
}
