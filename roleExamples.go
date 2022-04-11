package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func roleExamples() {
	var (
		filepath      = "./examples/role.yaml"
		name          = "test"
		labelSelector = "type=role"
		forceDelete   = false
	)
	role, err := k8s.NewRole(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = filepath
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = role

	// 1. create role
	role.Delete(name)
	if s, err := role.Create(filepath); err != nil {
		log.Error("create role failed")
		log.Error(err)
	} else {
		log.Infof("create role %s success.", s.Name)
	}
	// 2. update role
	if s, err := role.Update(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("update role %s success.", s.Name)
	}
	// 3. apply role
	if s, err := role.Apply(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply role %s success.", s.Name)
	}
	role.Delete(name)
	if s, err := role.Apply(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply role %s success.", s.Name)
	}
	// 4. delete role
	if err := role.Delete(name); err != nil {
		log.Error("delete role failed")
		log.Error(err)
	} else {
		log.Infof("delete role %s success.", name)
	}
	// 5. get role
	role.Create(filepath)
	if s, err := role.Get(name); err != nil {
		log.Error("get role failed")
		log.Error(err)
	} else {
		log.Infof("get role %s success.", s.Name)
	}
	// 6. list role
	if sl, err := role.List(labelSelector); err != nil {
		log.Error("list role failed")
		log.Error(err)
	} else {
		log.Info("list role success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7. watch role
	log.Info("start watch role")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			role.Apply(filepath)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(10)))
			//time.Sleep(time.Second * 125)
			role.Delete(name)
		}
	}()
	role.Watch(name,
		func(x interface{}) {
			log.Info("add role.")
		},
		func(x interface{}) {
			log.Info("modified role.")
		},
		func(x interface{}) {
			log.Info("deleted role.")
		},
		nil,
	)
}
