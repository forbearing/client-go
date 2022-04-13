package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func networkpolicyExamples() {
	var (
		yamlfile      = "./testData/networkpolicy.yaml"
		name          = "test"
		labelSelector = "type=networkpolicy"
		forceDelete   = false
	)
	networkpolicy, err := k8s.NewNetworkPolicy(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = networkpolicy

	// 1. create networkpolicy
	networkpolicy.Delete(name)
	if s, err := networkpolicy.Create(yamlfile); err != nil {
		log.Error("create networkpolicy failed")
		log.Error(err)
	} else {
		log.Infof("create networkpolicy %s success.", s.Name)
	}
	// 2. update networkpolicy
	if s, err := networkpolicy.Update(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("update networkpolicy %s success.", s.Name)
	}
	// 3. apply networkpolicy
	if s, err := networkpolicy.Apply(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply networkpolicy %s success.", s.Name)
	}
	networkpolicy.Delete(name)
	if s, err := networkpolicy.Apply(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply networkpolicy %s success.", s.Name)
	}
	// 4. delete networkpolicy
	if err := networkpolicy.Delete(name); err != nil {
		log.Error("delete networkpolicy failed")
		log.Error(err)
	} else {
		log.Infof("delete networkpolicy %s success.", name)
	}
	// 5. get networkpolicy
	networkpolicy.Create(yamlfile)
	if s, err := networkpolicy.Get(name); err != nil {
		log.Error("get networkpolicy failed")
		log.Error(err)
	} else {
		log.Infof("get networkpolicy %s success.", s.Name)
	}
	// 6. list networkpolicy
	if sl, err := networkpolicy.List(labelSelector); err != nil {
		log.Error("list networkpolicy failed")
		log.Error(err)
	} else {
		log.Info("list networkpolicy success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7. watch networkpolicy
	log.Info("start watch networkpolicy")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			networkpolicy.Apply(yamlfile)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(10)))
			//time.Sleep(time.Second * 125)
			networkpolicy.Delete(name)
		}
	}()
	networkpolicy.Watch(name,
		func(x interface{}) {
			log.Info("add networkpolicy.")
		},
		func(x interface{}) {
			log.Info("modified networkpolicy.")
		},
		func(x interface{}) {
			log.Info("deleted networkpolicy.")
		},
		"",
	)
}
