package main

import (
	"hybfkuf/pkg/k8s"
	"time"

	log "github.com/sirupsen/logrus"
)

func kubectl() {
	log.SetLevel(log.DebugLevel)
	var (
		filepath = "./examples/all.yaml"
		//filepath = "./examples/nginx-sts.yaml"
		err error
	)
	err = k8s.ApplyF(ctx, *kubeconfig, filepath)
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Second * 30)
	log.Info()
	err = k8s.DeleteF(ctx, *kubeconfig, filepath)
	if err != nil {
		log.Fatal(err)
	}
}
