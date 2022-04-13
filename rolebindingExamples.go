package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func rolebindingExamples() {
	var (
		yamlfile      = "./testData/rolebinding.yaml"
		name          = "test"
		labelSelector = "type=rolebinding"
		forceDelete   = false
	)
	rolebinding, err := k8s.NewRoleBinding(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = rolebinding

	// 1. create rolebinding
	rolebinding.Delete(name)
	if s, err := rolebinding.Create(yamlfile); err != nil {
		log.Error("create rolebinding failed")
		log.Error(err)
	} else {
		log.Infof("create rolebinding %s success.", s.Name)
	}
	// 2. update rolebinding
	if s, err := rolebinding.Update(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("update rolebinding %s success.", s.Name)
	}
	// 3. apply rolebinding
	if s, err := rolebinding.Apply(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply rolebinding %s success.", s.Name)
	}
	rolebinding.Delete(name)
	if s, err := rolebinding.Apply(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply rolebinding %s success.", s.Name)
	}
	// 4. delete rolebinding
	if err := rolebinding.Delete(name); err != nil {
		log.Error("delete rolebinding failed")
		log.Error(err)
	} else {
		log.Infof("delete rolebinding %s success.", name)
	}
	// 5. get rolebinding
	rolebinding.Create(yamlfile)
	if s, err := rolebinding.Get(name); err != nil {
		log.Error("get rolebinding failed")
		log.Error(err)
	} else {
		log.Infof("get rolebinding %s success.", s.Name)
	}
	// 6. list rolebinding
	if sl, err := rolebinding.List(labelSelector); err != nil {
		log.Error("list rolebinding failed")
		log.Error(err)
	} else {
		log.Info("list rolebinding success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7. watch rolebinding
	log.Info("start watch rolebinding")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			rolebinding.Apply(yamlfile)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(10)))
			//time.Sleep(time.Second * 125)
			rolebinding.Delete(name)
		}
	}()
	rolebinding.Watch(name,
		func(x interface{}) {
			log.Info("add rolebinding.")
		},
		func(x interface{}) {
			log.Info("modified rolebinding.")
		},
		func(x interface{}) {
			log.Info("deleted rolebinding.")
		},
		nil,
	)
}
