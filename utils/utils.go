package utils

import (
	"flag"
	"log"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func GetClientset() *kubernetes.Clientset {
	home := homedir.HomeDir()
	kubeconfig := flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "Location to the Kubeconfig file")

	/* Check whether code is running internally or externally and authenticate accordingly
	if kubeconfig exists {
		build config from kubeconfig
	} else {
		Build Config from SA credentials
	}
	*/
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Printf("Error Building config from Kubeconfig: %s\n", err.Error())

		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Error Building Config: %s\n", err.Error())
		}
	}
	flag.Parse()

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error Getting clientset: %s\n", err.Error())
	}

	return clientset
}
