package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func cronjobExamples() {
	var (
		filepath      = "./examples/cronjob.yaml"
		name          = "hello"
		labelSelector = "name=hello"
		forceDelete   = false
	)
	cronjob, err := k8s.NewCronJob(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = filepath
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = cronjob

	// 1. create cronjob
	cronjob.Delete(name)
	time.Sleep(time.Second * 3)
	if s, err := cronjob.Create(filepath); err != nil {
		log.Error("create cronjob failed")
		log.Error(err)
	} else {
		log.Infof("create cronjob %s success.", s.Name)
	}
	// 2. update cronjob
	if s, err := cronjob.Update(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("update cronjob %s success.", s.Name)
	}
	// 3. apply cronjob
	if s, err := cronjob.Apply(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply cronjob %s success.", s.Name)
	}
	cronjob.Delete(name)
	if s, err := cronjob.Apply(filepath); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply cronjob %s success.", s.Name)
	}
	// 4. delete cronjob
	if err := cronjob.Delete(name); err != nil {
		log.Error("delete cronjob failed")
		log.Error(err)
	} else {
		log.Infof("delete cronjob %s success.", name)
	}
	time.Sleep(time.Second * 3)
	// 5. get cronjob
	cronjob.Create(filepath)
	if s, err := cronjob.Get(name); err != nil {
		log.Error("get cronjob failed")
		log.Error(err)
	} else {
		log.Infof("get cronjob %s success.", s.Name)
	}
	// 6. list cronjob
	if sl, err := cronjob.List(labelSelector); err != nil {
		log.Error("list cronjob failed")
		log.Error(err)
	} else {
		log.Info("list cronjob success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7. watch cronjob
	log.Info("start watch cronjob")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			cronjob.Apply(filepath)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			//time.Sleep(time.Second * time.Duration(rand.Intn(15)))
			time.Sleep(time.Second * 125)
			cronjob.Delete(name)
		}
	}()
	cronjob.Watch(name,
		func(x interface{}) {
			log.Info("add cronjob.")
		},
		func(x interface{}) {
			log.Info("modified cronjob.")
		},
		func(x interface{}) {
			log.Info("deleted cronjob.")
		},
		nil,
	)
}
