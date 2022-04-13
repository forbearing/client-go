package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func namespaceExamples() {
	var (
		yamlfile      = "./testData/NAMESPACE.yaml"
		name          = "test1"
		labelSelector = "type=NAMESPACE"
		forceDelete   = false
	)
	namespace, err := k8s.NewNamespace(ctx, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = NAMESPACE

	// 1. create namespace
	namespace.Delete(name)
	if s, err := namespace.Create(yamlfile); err != nil {
		log.Error("create namespace failed")
		log.Error(err)
	} else {
		log.Infof("create namespace %s success.", s.Name)
	}
	// 2. update namespace
	if s, err := namespace.Update(yamlfile); err != nil {
		log.Error("update namespace failed")
		log.Error(err)
	} else {
		log.Infof("update namespace %s success.", s.Name)
	}
	// 3. apply namespace
	if s, err := namespace.Apply(yamlfile); err != nil {
		log.Error("apply namespace failed")
		log.Error(err)
	} else {
		log.Infof("apply namespace %s success.", s.Name)
	}
	namespace.Delete(name)
	if s, err := namespace.Apply(yamlfile); err != nil {
		log.Error("apply namespace failed")
		log.Error(err)
	} else {
		log.Infof("apply namespace %s success.", s.Name)
	}
	// 4. delete namespace
	if err := namespace.Delete(name); err != nil {
		log.Error("delete namespace failed")
		log.Error(err)
	} else {
		log.Infof("delete namespace %s success.", name)
	}
	// 5. get namespace
	namespace.Create(yamlfile)
	if s, err := namespace.Get(name); err != nil {
		log.Error("get namespace failed")
		log.Error(err)
	} else {
		log.Infof("get namespace %s success.", s.Name)
	}
	// 6. list namespace
	if sl, err := namespace.List(labelSelector); err != nil {
		log.Error("list namespace failed")
		log.Error(err)
	} else {
		log.Info("list namespace success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7. watch namespace
	log.Info("start watch namespace")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			namespace.Apply(yamlfile)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			//time.Sleep(time.Second * time.Duration(rand.Intn(30)))
			time.Sleep(time.Second * 20)
			namespace.Delete(name)
		}
	}()
	namespace.Watch(name,
		func(x interface{}) {
			log.Info("add namespace.")
		},
		func(x interface{}) {
			log.Info("modified namespace.")
		},
		func(x interface{}) {
			log.Info("deleted namespace.")
		},
		nil,
	)
}
