package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
)

func cronjobExamples() {
	var (
		yamlfile      = "./testData/cronjob.yaml"
		name          = "hello"
		labelSelector = "name=hello"
		jobs          []batchv1.Job
	)
	cjHandler, err := k8s.NewCronJob(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = cjHandler

	// 1. create cronjob
	cjHandler.Delete(name)
	if cj, err := cjHandler.Create(yamlfile); err != nil {
		log.Error("create cronjob failed")
		log.Error(err)
	} else {
		log.Infof("create cronjob %q success.", cj.Name)
	}
	// 2. update cronjob
	if cj, err := cjHandler.Update(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("update cronjob %q success.", cj.Name)
	}
	// 3. apply cronjob
	if cj, err := cjHandler.Apply(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply cronjob %q success.", cj.Name)
	}
	cjHandler.Delete(name)
	if cj, err := cjHandler.Apply(yamlfile); err != nil {
		log.Error(err)
	} else {
		log.Infof("apply cronjob %q success.", cj.Name)
	}
	// 4. delete cronjob
	if err := cjHandler.Delete(name); err != nil {
		log.Error("delete cronjob failed")
		log.Error(err)
	} else {
		log.Infof("delete cronjob %q success.", name)
	}
	// 4. delete cronjob from file
	cjHandler.Apply(yamlfile)
	if err := cjHandler.DeleteFromFile(yamlfile); err != nil {
		log.Error("delete cronjob from file failed")
		log.Error(err)
	} else {
		log.Infof("delete cronjob %q from file success.", name)
	}
	// 5. get cronjob
	cjHandler.Create(yamlfile)
	if cj, err := cjHandler.Get(name); err != nil {
		log.Error("get cronjob failed")
		log.Error(err)
	} else {
		log.Infof("get cronjob %q success.", cj.Name)
	}
	// 6. list cronjob
	if cjList, err := cjHandler.List(labelSelector); err != nil {
		log.Error("list cronjob failed")
		log.Error(err)
	} else {
		log.Info("list cronjob success.")
		for _, cj := range cjList.Items {
			log.Info(cj.Name)
		}
	}

	// 7. get cronjob details
	log.Info()
	getJobs := func() {
		if jobs, err = cjHandler.GetJobs(name); err != nil {
			log.Error("get cronjob generated job failed")
			log.Error(err)
		} else {
			log.Infof("get cronjob %q generated job success.", name)
			for _, job := range jobs {
				log.Info(job.Name)
			}
		}
	}
	getJobs()

	// 8. watch cronjob
	log.Info()
	log.Info("start watch cronjob")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			cjHandler.Apply(yamlfile)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			//time.Sleep(time.Second * time.Duration(rand.Intn(15)))
			time.Sleep(time.Second * 390)
			cjHandler.Delete(name)
		}
	}()
	cjHandler.Watch(name,
		func(x interface{}) {
			log.Info("added cronjob.")
		},
		func(x interface{}) {
			log.Info("modified cronjob.")
			getJobs()
		},
		func(x interface{}) {
			log.Info("deleted cronjob.")
		},
		nil,
	)
}
