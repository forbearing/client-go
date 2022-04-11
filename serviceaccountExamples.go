package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func serviceaccountExamples() {
	var (
		filepath      = "./examples/serviceaccount.yaml"
		name          = "test"
		labelSelector = "type=serviceaccount"
		forceDelete   = false
	)
	serviceaccount, err := k8s.NewServiceAccount(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = filepath
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = serviceaccount

	// 1. create serviceaccount
	serviceaccount.Delete(name)
	if s, err := serviceaccount.Create(filepath); err != nil {
		log.Error("create serviceaccount failed")
		log.Error(err)
	} else {
		log.Infof("create serviceaccount %s success.", s.Name)
	}
	// 2. update serviceaccount
	if s, err := serviceaccount.Update(filepath); err != nil {
		log.Error("update serviceaccount failed")
		log.Error(err)
	} else {
		log.Infof("update serviceaccount %s success.", s.Name)
	}
	// 3. apply serviceaccount
	if s, err := serviceaccount.Apply(filepath); err != nil {
		log.Error("apply serviceaccount failed")
		log.Error(err)
	} else {
		log.Infof("apply serviceaccount %s success.", s.Name)
	}
	serviceaccount.Delete(name)
	if s, err := serviceaccount.Apply(filepath); err != nil {
		log.Error("apply serviceaccount failed")
		log.Error(err)
	} else {
		log.Infof("apply serviceaccount %s success.", s.Name)
	}
	// 4. delete serviceaccount
	if err := serviceaccount.Delete(name); err != nil {
		log.Error("delete serviceaccount failed")
		log.Error(err)
	} else {
		log.Infof("delete serviceaccount %s success.", name)
	}
	// 5. get serviceaccount
	serviceaccount.Create(filepath)
	if s, err := serviceaccount.Get(name); err != nil {
		log.Error("get serviceaccount failed")
		log.Error(err)
	} else {
		log.Infof("get serviceaccount %s success.", s.Name)
	}
	// 6. list serviceaccount
	if sl, err := serviceaccount.List(labelSelector); err != nil {
		log.Error("list serviceaccount failed")
		log.Error(err)
	} else {
		log.Info("list serviceaccount success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7. watch serviceaccount
	log.Info("start watch serviceaccount")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			serviceaccount.Apply(filepath)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			//time.Sleep(time.Second * time.Duration(rand.Intn(30)))
			time.Sleep(time.Second * 30)
			serviceaccount.Delete(name)
		}
	}()
	serviceaccount.Watch(name,
		func(x interface{}) {
			log.Info("add serviceaccount.")
		},
		func(x interface{}) {
			log.Info("modified serviceaccount.")
		},
		func(x interface{}) {
			log.Info("deleted serviceaccount.")
		},
		nil,
	)
}
