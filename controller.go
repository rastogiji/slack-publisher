package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/slack-go/slack"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	v1 "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	queue       workqueue.RateLimitingInterface
	clientset   *kubernetes.Clientset
	depInformer v1.DeploymentInformer
}

func NewController(cs *kubernetes.Clientset, di v1.DeploymentInformer) *Controller {
	c := &Controller{
		clientset:   cs,
		depInformer: di,
		queue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "slack-publisher"),
	}
	di.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.addFunc,
	})

	return c
}

func (c *Controller) Run(ch chan struct{}) {
	log.Println("Starting Controller")
	if !cache.WaitForCacheSync(ch, c.depInformer.Informer().HasSynced) {
		log.Println("Waiting for Cache to Sync")
	}

	go wait.Until(c.runWorker, 1*time.Second, ch)
	<-ch
}

func (c *Controller) runWorker() {
	for c.processItem() {

	}
}

func (c *Controller) processItem() bool {

	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Forget(item)

	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		return false
	}

	err = c.syncHandler(key)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
		return false
	}
	return true
}

func (c *Controller) syncHandler(key string) error {
	ctx := context.Background()
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	if ns == "kube-system" || ns == "gmp-system" || ns == "gmp-public" || ns == "gke-mcs" {
		return nil
	}

	_, err = c.clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			fmt.Println("Object no longer exists in the cluster or is deleted")
			return nil
		}
		return err
	}
	err = c.slackPublisher(ctx, ns, name)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
	}
	return nil
}

func (c *Controller) slackPublisher(ctx context.Context, ns string, name string) error {
	var crs []corev1.Container
	dep, err := c.depInformer.Lister().Deployments(ns).Get(name)
	if err != nil {
		return err
	}
	depCopy := dep.DeepCopy()
	for _, cr := range depCopy.Spec.Template.Spec.Containers {
		if cr.Resources.Requests == nil || cr.Resources.Limits == nil {
			crs = append(crs, cr)
		}
	}
	if len(crs) == 0 {
		return nil
	}
	err = sendSlackAlert(crs, depCopy.Name, ns)
	if err != nil {
		return err
	}
	return nil
}

func sendSlackAlert(crs []corev1.Container, depName string, ns string) error {
	token := os.Getenv("TOKEN")
	channel := os.Getenv("CHANNEL")
	api := slack.New(token)
	msg := slackMessage(crs, depName, ns)

	log.Printf("Sending Message to Slack")
	_, _, err := api.PostMessage(channel, slack.MsgOptionText(msg, true))
	if err != nil {
		return errors.New("Error Sending Slack Message: " + err.Error())
	}
	return nil
}

func slackMessage(crs []corev1.Container, depName string, ns string) string {
	message := fmt.Sprintf("Following containers of Deployment *%s* in namespace *%s* do not have resource requests set\n", depName, ns)

	for i, cr := range crs {
		line := fmt.Sprintf("(%d). *Container Name*: %s\n", (i + 1), cr.Name)
		message = message + line
	}
	return message
}
func (c *Controller) addFunc(obj interface{}) {
	log.Println("Adding to the Queue")
	c.queue.Add(obj)
}
