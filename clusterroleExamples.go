package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func clusterroleExamples() {
	var (
		filepath      = "./examples/clusterrole.yaml"
		name          = "test"
		labelSelector = "type=clusterrole"
		forceDelete   = false
	)
	clusterrole, err := k8s.NewClusterRole(ctx, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = filepath
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = clusterrole

	// 1. create clusterrole
	clusterrole.Delete(name)
	if s, err := clusterrole.Create(filepath); err != nil {
		log.Error("create clusterrole failed")
		log.Error(err)
	} else {
		log.Infof("create clusterrole %s success.", s.Name)
	}
	// 2. update clusterrole
	if s, err := clusterrole.Update(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("update clusterrole %s success.", s.Name)
	}
	// 3. apply clusterrole
	if s, err := clusterrole.Apply(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply clusterrole %s success.", s.Name)
	}
	clusterrole.Delete(name)
	if s, err := clusterrole.Apply(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply clusterrole %s success.", s.Name)
	}
	// 4. delete clusterrole
	if err := clusterrole.Delete(name); err != nil {
		log.Error("delete clusterrole failed")
		log.Error(err)
	} else {
		log.Infof("delete clusterrole %s success.", name)
	}
	// 5. get clusterrole
	clusterrole.Create(filepath)
	if s, err := clusterrole.Get(name); err != nil {
		log.Error("get clusterrole failed")
		log.Error(err)
	} else {
		log.Infof("get clusterrole %s success.", s.Name)
	}
	// 6. list clusterrole
	if sl, err := clusterrole.List(labelSelector); err != nil {
		log.Error("list clusterrole failed")
		log.Error(err)
	} else {
		log.Info("list clusterrole success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7. watch clusterrole
	log.Info("start watch clusterrole")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			clusterrole.Apply(filepath)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(10)))
			//time.Sleep(time.Second * 125)
			clusterrole.Delete(name)
		}
	}()
	clusterrole.Watch(name,
		func(x interface{}) {
			log.Info("add clusterrole.")
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
