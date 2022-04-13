package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func clusterrolebindingExamples() {
	var (
		yamlfile      = "./testData/clusterrolebinding.yaml"
		name          = "test"
		labelSelector = "type=clusterrolebinding"
	)
	crbHandler, err := k8s.NewClusterRoleBinding(ctx, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = crbHandler

	// 1. create clusterrolebinding
	crbHandler.Delete(name)
	if crb, err := crbHandler.Create(yamlfile); err != nil {
		log.Error("create clusterrolebinding failed")
		log.Error(err)
	} else {
		log.Infof("create clusterrolebinding %q success.", crb.Name)
	}
	// 2. update clusterrolebinding
	if crb, err := crbHandler.Update(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("update clusterrolebinding %q success.", crb.Name)
	}
	// 3. apply clusterrolebinding
	if crb, err := crbHandler.Apply(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply clusterrolebinding %q success.", crb.Name)
	}
	crbHandler.Delete(name)
	if crb, err := crbHandler.Apply(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply clusterrolebinding %q success.", crb.Name)
	}
	// 4. delete clusterrolebinding
	if err := crbHandler.Delete(name); err != nil {
		log.Error("delete clusterrolebinding failed")
		log.Error(err)
	} else {
		log.Infof("delete clusterrolebinding %q success.", name)
	}
	// 5. delete clusterrolebinding from file
	crbHandler.Apply(yamlfile)
	if err := crbHandler.DeleteFromFile(yamlfile); err != nil {
		log.Error("delete clusterrolebinding from file failed")
		log.Error(err)
	} else {
		log.Infof("delete clusterrolebinding %q from file success.", name)
	}
	// 6. get clusterrolebinding
	crbHandler.Create(yamlfile)
	if crb, err := crbHandler.Get(name); err != nil {
		log.Error("get clusterrolebinding failed")
		log.Error(err)
	} else {
		log.Infof("get clusterrolebinding %q success.", crb.Name)
	}
	// 7. list clusterrolebinding
	if crbList, err := crbHandler.List(labelSelector); err != nil {
		log.Error("list clusterrolebinding failed")
		log.Error(err)
	} else {
		log.Info("list clusterrolebinding success.")
		for _, crb := range crbList.Items {
			log.Info(crb.Name)
		}
	}
	// 8. watch clusterrolebinding
	log.Info("start watch clusterrolebinding")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			crbHandler.Apply(yamlfile)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(10)))
			crbHandler.Delete(name)
		}
	}()
	crbHandler.Watch(name,
		func(x interface{}) {
			log.Info("added clusterrolebinding.")
		},
		func(x interface{}) {
			log.Info("modified clusterrolebinding.")
		},
		func(x interface{}) {
			log.Info("deleted clusterrolebinding.")
		},
		nil,
	)
}
