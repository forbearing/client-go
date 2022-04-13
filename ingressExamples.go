package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func ingressExamples() {
	var (
		yamlfile      = "./testData/ingress.yaml"
		name          = "test"
		labelSelector = "type=ingress"
		forceDelete   = false
	)
	ingress, err := k8s.NewIngress(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = ingress

	// 1. create ingress
	ingress.Delete(name)
	if s, err := ingress.Create(yamlfile); err != nil {
		log.Error("create ingress failed")
		log.Error(err)
	} else {
		log.Infof("create ingress %s success.", s.Name)
	}
	// 2. update ingress
	if s, err := ingress.Update(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("update ingress %s success.", s.Name)
	}
	// 3. apply ingress
	if s, err := ingress.Apply(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply ingress %s success.", s.Name)
	}
	ingress.Delete(name)
	if s, err := ingress.Apply(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply ingress %s success.", s.Name)
	}
	// 4. delete ingress
	if err := ingress.Delete(name); err != nil {
		log.Error("delete ingress failed")
		log.Error(err)
	} else {
		log.Infof("delete ingress %s success.", name)
	}
	// 5. get ingress
	ingress.Create(yamlfile)
	if s, err := ingress.Get(name); err != nil {
		log.Error("get ingress failed")
		log.Error(err)
	} else {
		log.Infof("get ingress %s success.", s.Name)
	}
	// 6. list ingress
	if sl, err := ingress.List(labelSelector); err != nil {
		log.Error("list ingress failed")
		log.Error(err)
	} else {
		log.Info("list ingress success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7. watch ingress
	log.Info("start watch ingress")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			ingress.Apply(yamlfile)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(10)))
			//time.Sleep(time.Second * 125)
			ingress.Delete(name)
		}
	}()
	ingress.Watch(name,
		func(x interface{}) {
			log.Info("add ingress.")
		},
		func(x interface{}) {
			log.Info("modified ingress.")
		},
		func(x interface{}) {
			log.Info("deleted ingress.")
		},
		nil,
	)
}
