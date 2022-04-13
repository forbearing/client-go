package main

import (
	"hybfkuf/pkg/k8s"
	"time"

	log "github.com/sirupsen/logrus"
)

func kubectl() {
	log.SetLevel(log.DebugLevel)
	var (
		yamlfile = "./testData/all.yaml"
		//yamlfile = "./testData/nginx-sts.yaml"
		err error
	)
	err = k8s.ApplyF(ctx, *kubeconfig, yamlfile)
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Second * 30)
	log.Info()
	err = k8s.DeleteF(ctx, *kubeconfig, yamlfile)
	if err != nil {
		log.Fatal(err)
	}
}
