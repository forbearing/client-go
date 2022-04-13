package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func clusterroleExamples() {
	var (
		yamlfile      = "./testData/clusterrole.yaml"
		name          = "test"
		labelSelector = "type=clusterrole"
	)
	crHandler, err := k8s.NewClusterRole(ctx, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = crHandler

	// 1. create clusterrole
	crHandler.Delete(name)
	if cr, err := crHandler.Create(yamlfile); err != nil {
		log.Error("create clusterrole failed")
		log.Error(err)
	} else {
		log.Infof("create clusterrole %q success.", cr.Name)
	}
	// 2. update clusterrole
	if cr, err := crHandler.Update(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("update clusterrole %q success.", cr.Name)
	}
	// 3. apply clusterrole
	if cr, err := crHandler.Apply(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply clusterrole %q success.", cr.Name)
	}
	crHandler.Delete(name)
	if cr, err := crHandler.Apply(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply clusterrole %q success.", cr.Name)
	}
	// 4. delete clusterrole
	if err := crHandler.Delete(name); err != nil {
		log.Error("delete clusterrole failed")
		log.Error(err)
	} else {
		log.Infof("delete clusterrole %q success.", name)
	}
	// 5. delete clusterrole from file
	crHandler.Apply(yamlfile)
	if err := crHandler.DeleteFromFile(yamlfile); err != nil {
		log.Error("delete clusterrole from file failed")
		log.Error(err)
	} else {
		log.Infof("delete clusterrole %q from file success.", name)
	}
	// 6. get clusterrole
	crHandler.Create(yamlfile)
	if cr, err := crHandler.Get(name); err != nil {
		log.Error("get clusterrole failed")
		log.Error(err)
	} else {
		log.Infof("get clusterrole %q success.", cr.Name)
	}
	// 7. list clusterrole
	if crList, err := crHandler.List(labelSelector); err != nil {
		log.Error("list clusterrole failed")
		log.Error(err)
	} else {
		log.Info("list clusterrole success.")
		for _, cr := range crList.Items {
			log.Info(cr.Name)
		}
	}
	// 8. watch clusterrole
	log.Info("start watch clusterrole")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			crHandler.Apply(yamlfile)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(10)))
			//time.Sleep(time.Second * 125)
			crHandler.Delete(name)
		}
	}()
	crHandler.Watch(name,
		func(x interface{}) {
			log.Info("added clusterrole.")
		},
		func(x interface{}) {
			log.Info("modified clusterrole.")
		},
		func(x interface{}) {
			log.Info("deleted clusterrole.")
		},
		nil,
	)
}
