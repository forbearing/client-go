package main

import (
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	namespace = "default"
)

func main() {
	clientset, err := NewClient()
	if err != nil {
		logrus.Error(err)
		return
	}
	_ = clientset
}

func NewClient() (clientset *kubernetes.Clientset, err error) {
	clientCfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err = kubernetes.NewForConfig(clientCfg)
	if err != nil {
		return nil, err
	}

	return
}
