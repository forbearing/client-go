package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func configmapExamples() {
	var (
		filepath      = "./examples/configmap.yaml"
		name          = "test"
		labelSelector = "type=configmap"
		forceDelete   = false
	)
	configmap, err := k8s.NewConfigMap(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = filepath
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = configmap

	// 1. create configmap
	configmap.Delete(name)
	if s, err := configmap.Create(filepath); err != nil {
		log.Error("create configmap failed")
		log.Error(err)
	} else {
		log.Infof("create configmap %s success.", s.Name)
	}
	// 2. update configmap
	if s, err := configmap.Update(filepath); err != nil {
		log.Error("update configmap failed")
		log.Error(err)
	} else {
		log.Infof("update configmap %s success.", s.Name)
	}
	// 3. apply configmap
	if s, err := configmap.Apply(filepath); err != nil {
		log.Error("apply configmap failed")
		log.Error(err)
	} else {
		log.Infof("apply configmap %s success.", s.Name)
	}
	configmap.Delete(name)
	if s, err := configmap.Apply(filepath); err != nil {
		log.Error("apply configmap failed")
		log.Error(err)
	} else {
		log.Infof("apply configmap %s success.", s.Name)
	}
	// 4. delete configmap
	if err := configmap.Delete(name); err != nil {
		log.Error("delete configmap failed")
		log.Error(err)
	} else {
		log.Infof("delete configmap %s success.", name)
	}
	// 5. get configmap
	configmap.Create(filepath)
	if s, err := configmap.Get(name); err != nil {
		log.Error("get configmap failed")
		log.Error(err)
	} else {
		log.Infof("get configmap %s success.", s.Name)
	}
	// 6. list configmap
	if sl, err := configmap.List(labelSelector); err != nil {
		log.Error("list configmap failed")
		log.Error(err)
	} else {
		log.Info("list configmap success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7 . get configmap data
	if data, err := configmap.GetData(name); err != nil {
		log.Error("get configmap data failed")
		log.Error(err)
	} else {
		log.Info("get configmap data success.")
		for key, value := range data {
			log.Infof("file name: %s", key)
			log.Infof("file data:\n%s", value)
		}
	}
	// 8. watch configmap
	log.Info("start watch configmap")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			configmap.Apply(filepath)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			//time.Sleep(time.Second * time.Duration(rand.Intn(30)))
			time.Sleep(time.Second * 10)
			configmap.Delete(name)
		}
	}()
	configmap.Watch(name,
		func(x interface{}) {
			log.Info("add configmap.")
		},
		func(x interface{}) {
			log.Info("modified configmap.")
		},
		func(x interface{}) {
			log.Info("deleted configmap.")
		},
		nil,
	)
}
