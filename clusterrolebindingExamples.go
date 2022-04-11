package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func clusterrolebindingExamples() {
	var (
		filepath      = "./examples/clusterrolebinding.yaml"
		name          = "test"
		labelSelector = "type=clusterrolebinding"
		forceDelete   = false
	)
	clusterrolebinding, err := k8s.NewClusterRoleBinding(ctx, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = filepath
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = clusterrolebinding

	// 1. create clusterrolebinding
	clusterrolebinding.Delete(name)
	if s, err := clusterrolebinding.Create(filepath); err != nil {
		log.Error("create clusterrolebinding failed")
		log.Error(err)
	} else {
		log.Infof("create clusterrolebinding %s success.", s.Name)
	}
	// 2. update clusterrolebinding
	if s, err := clusterrolebinding.Update(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("update clusterrolebinding %s success.", s.Name)
	}
	// 3. apply clusterrolebinding
	if s, err := clusterrolebinding.Apply(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply clusterrolebinding %s success.", s.Name)
	}
	clusterrolebinding.Delete(name)
	if s, err := clusterrolebinding.Apply(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply clusterrolebinding %s success.", s.Name)
	}
	// 4. delete clusterrolebinding
	if err := clusterrolebinding.Delete(name); err != nil {
		log.Error("delete clusterrolebinding failed")
		log.Error(err)
	} else {
		log.Infof("delete clusterrolebinding %s success.", name)
	}
	// 5. get clusterrolebinding
	clusterrolebinding.Create(filepath)
	if s, err := clusterrolebinding.Get(name); err != nil {
		log.Error("get clusterrolebinding failed")
		log.Error(err)
	} else {
		log.Infof("get clusterrolebinding %s success.", s.Name)
	}
	// 6. list clusterrolebinding
	if sl, err := clusterrolebinding.List(labelSelector); err != nil {
		log.Error("list clusterrolebinding failed")
		log.Error(err)
	} else {
		log.Info("list clusterrolebinding success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7. watch clusterrolebinding
	log.Info("start watch clusterrolebinding")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			clusterrolebinding.Apply(filepath)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(10)))
			//time.Sleep(time.Second * 125)
			clusterrolebinding.Delete(name)
		}
	}()
	clusterrolebinding.Watch(name,
		func(x interface{}) {
			log.Info("add clusterrolebinding.")
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
