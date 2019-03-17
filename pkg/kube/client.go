package kube

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

type Client interface {
	Get() kubernetes.Interface
}

type client struct {

}

func NewClient() Client {
	return &client{}
}

func (c *client) Get() kubernetes.Interface {
	cfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %s", err)
	}

	return kubeClient
}
