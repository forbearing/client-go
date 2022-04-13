package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func configmapExamples() {
	var (
		yamlfile      = "./testData/configmap.yaml"
		name          = "test"
		labelSelector = "type=configmap"
	)
	cmHandler, err := k8s.NewConfigMap(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = cmHandler

	// 1. create configmap
	cmHandler.Delete(name)
	if cm, err := cmHandler.Create(yamlfile); err != nil {
		log.Error("create configmap failed")
		log.Error(err)
	} else {
		log.Infof("create configmap %q success.", cm.Name)
	}
	// 2. update configmap
	if cm, err := cmHandler.Update(yamlfile); err != nil {
		log.Error("update configmap failed")
		log.Error(err)
	} else {
		log.Infof("update configmap %q success.", cm.Name)
	}
	// 3. apply configmap
	if cm, err := cmHandler.Apply(yamlfile); err != nil {
		log.Error("apply configmap failed")
		log.Error(err)
	} else {
		log.Infof("apply configmap %q success.", cm.Name)
	}
	cmHandler.Delete(name)
	if cm, err := cmHandler.Apply(yamlfile); err != nil {
		log.Error("apply configmap failed")
		log.Error(err)
	} else {
		log.Infof("apply configmap %q success.", cm.Name)
	}
	// 4. delete configmap
	if err := cmHandler.Delete(name); err != nil {
		log.Error("delete configmap failed")
		log.Error(err)
	} else {
		log.Infof("delete configmap %q success.", name)
	}
	// 5. delete configmap from file
	cmHandler.Apply(yamlfile)
	if err := cmHandler.DeleteFromFile(yamlfile); err != nil {
		log.Error("delete configmap from file failed")
		log.Error(err)
	} else {
		log.Infof("delete configmap %q from file success.", name)
	}
	// 6. get configmap
	cmHandler.Create(yamlfile)
	if cm, err := cmHandler.Get(name); err != nil {
		log.Error("get configmap failed")
		log.Error(err)
	} else {
		log.Infof("get configmap %q success.", cm.Name)
	}
	// 7. list configmap
	if cmList, err := cmHandler.List(labelSelector); err != nil {
		log.Error("list configmap failed")
		log.Error(err)
	} else {
		log.Info("list configmap success.")
		for _, cm := range cmList.Items {
			log.Info(cm.Name)
		}
	}

	// 8. get configmap data
	if data, err := cmHandler.GetData(name); err != nil {
		log.Error("get configmap data failed")
		log.Error(err)
	} else {
		log.Info("get configmap data success.")
		for key, value := range data {
			log.Infof("file name: %q", key)
			log.Infof("file data: %q", value)
		}
	}

	// 9. watch configmap
	log.Info("start watch configmap")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			cmHandler.Apply(yamlfile)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			//time.Sleep(time.Second * time.Duration(rand.Intn(30)))
			time.Sleep(time.Second * 10)
			cmHandler.Delete(name)
		}
	}()
	cmHandler.Watch(name,
		func(x interface{}) {
			log.Info("added cmHandler.")
		},
		func(x interface{}) {
			log.Info("modified cmHandler.")
		},
		func(x interface{}) {
			log.Info("deleted cmHandler.")
		},
		nil,
	)
}
