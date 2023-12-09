package main

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/rastogij/slack_publisher/utils"
	"k8s.io/client-go/informers"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading the Env variables: %s\n", err.Error())
	}
	cs := utils.GetClientset()

	ch := make(chan struct{})
	informer := informers.NewSharedInformerFactory(cs, 10*time.Minute)
	depInformer := informer.Apps().V1().Deployments()
	c := NewController(cs, depInformer)
	informer.Start(ch)
	c.Run(ch)
}
