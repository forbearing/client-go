package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func serviceExamples() {
	var (
		yamlfile      = "./testData/service.yaml"
		name          = "test"
		labelSelector = "type=service"
		forceDelete   = false
	)
	service, err := k8s.NewService(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = service

	// 1. create service
	service.Delete(name)
	if s, err := service.Create(yamlfile); err != nil {
		log.Error("create service failed")
		log.Error(err)
	} else {
		log.Infof("create service %s success.", s.Name)
	}
	// 2. update service
	if s, err := service.Update(yamlfile); err != nil {
		log.Error("update service failed")
		log.Error(err)
	} else {
		log.Infof("update service %s success.", s.Name)
	}
	// 3. apply service
	if s, err := service.Apply(yamlfile); err != nil {
		log.Error("apply service failed")
		log.Error(err)
	} else {
		log.Infof("apply service %s success.", s.Name)
	}
	service.Delete(name)
	if s, err := service.Apply(yamlfile); err != nil {
		log.Error("apply service failed")
		log.Error(err)
	} else {
		log.Infof("apply service %s success.", s.Name)
	}
	// 4. delete service
	if err := service.Delete(name); err != nil {
		log.Error("delete service failed")
		log.Error(err)
	} else {
		log.Infof("delete service %s success.", name)
	}
	// 5. get service
	service.Create(yamlfile)
	if s, err := service.Get(name); err != nil {
		log.Error("get service failed")
		log.Error(err)
	} else {
		log.Infof("get service %s success.", s.Name)
	}
	// 6. list service
	if sl, err := service.List(labelSelector); err != nil {
		log.Error("list service failed")
		log.Error(err)
	} else {
		log.Info("list service success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7. watch service
	log.Info("start watch service")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			service.Apply(yamlfile)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			//time.Sleep(time.Second * time.Duration(rand.Intn(30)))
			time.Sleep(time.Second * 10)
			service.Delete(name)
		}
	}()
	service.Watch(name,
		func(x interface{}) {
			log.Info("add service.")
		},
		func(x interface{}) {
			log.Info("modified service.")
		},
		func(x interface{}) {
			log.Info("deleted service.")
		},
		nil,
	)
}
