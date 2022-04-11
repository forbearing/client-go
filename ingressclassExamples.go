package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func ingressclassExamples() {
	var (
		filepath      = "./examples/ingressclass.yaml"
		name          = "test"
		labelSelector = "type=ingressclass"
		forceDelete   = false
	)
	ingressclass, err := k8s.NewIngressClass(ctx, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = filepath
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = ingressclass

	// 4. delete ingressclass
	if err := ingressclass.Delete(name); err != nil {
		log.Error("delete ingressclass failed")
		log.Error(err)
	} else {
		log.Infof("delete ingressclass %s success.", name)
	}

	// 1. create ingressclass
	ingressclass.Delete(name)
	if s, err := ingressclass.Create(filepath); err != nil {
		log.Error("create ingressclass failed")
		log.Error(err)
	} else {
		log.Infof("create ingressclass %s success.", s.Name)
	}
	// 2. update ingressclass
	if s, err := ingressclass.Update(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("update ingressclass %s success.", s.Name)
	}
	// 3. apply ingressclass
	if s, err := ingressclass.Apply(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply ingressclass %s success.", s.Name)
	}
	ingressclass.Delete(name)
	if s, err := ingressclass.Apply(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply ingressclass %s success.", s.Name)
	}
	// 4. delete ingressclass
	if err := ingressclass.Delete(name); err != nil {
		log.Error("delete ingressclass failed")
		log.Error(err)
	} else {
		log.Infof("delete ingressclass %s success.", name)
	}
	// 5. get ingressclass
	ingressclass.Create(filepath)
	if s, err := ingressclass.Get(name); err != nil {
		log.Error("get ingressclass failed")
		log.Error(err)
	} else {
		log.Infof("get ingressclass %s success.", s.Name)
	}
	// 6. list ingressclass
	if sl, err := ingressclass.List(labelSelector); err != nil {
		log.Error("list ingressclass failed")
		log.Error(err)
	} else {
		log.Info("list ingressclass success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7. watch ingressclass
	log.Info("start watch ingressclass")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			ingressclass.Apply(filepath)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(10)))
			//time.Sleep(time.Second * 125)
			ingressclass.Delete(name)
		}
	}()
	ingressclass.Watch(name,
		func(x interface{}) {
			log.Info("add ingressclass.")
		},
		func(x interface{}) {
			log.Info("modified ingressclass.")
		},
		func(x interface{}) {
			log.Info("deleted ingressclass.")
		},
		nil,
	)
}
