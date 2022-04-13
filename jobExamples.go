package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func jobExamples() {
	var (
		yamlfile      = "./testData/job.yaml"
		name          = "echo"
		labelSelector = "job-name=echo"
	)
	jobHandler, err := k8s.NewJob(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = jobHandler

	// 1. create job
	jobHandler.SetPropagationPolicy("background")
	jobHandler.SetForceDelete(true)
	jobHandler.Delete(name)
	time.Sleep(time.Second * 5)
	if job, err := jobHandler.Create(yamlfile); err != nil {
		log.Error("create job failed")
		log.Error(err)
	} else {
		log.Infof("create job %q success.", job.Name)
	}
	//// 2. update job
	//if job, err := jobHandler.Update(yamlfile); err != nil {
	//    log.Error(err)
	//} else {
	//    log.Infof("update job %q success.", job.Name)
	//}
	//// 3. apply job
	//if job, err := jobHandler.Apply(yamlfile); err != nil {
	//    log.Error(err)
	//} else {
	//    log.Infof("apply job %q success.", job.Name)
	//}
	//jobHandler.Delete(name, forceDelete)
	//if job, err := jobHandler.Apply(yamlfile); err != nil {
	//    log.Error(err)
	//} else {
	//    log.Infof("apply job %q success.", job.Name)
	//}
	// 4. delete job
	if err := jobHandler.Delete(name); err != nil {
		log.Error("delete job failed")
		log.Error(err)
	} else {
		log.Infof("delete job %q success.", name)
	}
	// 5. delete job from file
	time.Sleep(time.Second * 5)
	jobHandler.Apply(yamlfile)
	if err := jobHandler.DeleteFromFile(yamlfile); err != nil {
		log.Error("delete job from file failed")
		log.Error(err)
	} else {
		log.Infof("delete job from file %q success.", name)
	}
	// 6. get job
	jobHandler.Create(yamlfile)
	if job, err := jobHandler.Get(name); err != nil {
		log.Error("get job failed")
		log.Error(err)
	} else {
		log.Infof("get job %q success.", job.Name)
	}
	// 7. list job
	if jobList, err := jobHandler.List(labelSelector); err != nil {
		log.Error("list job failed")
		log.Error(err)
	} else {
		log.Info("list job success.")
		for _, job := range jobList.Items {
			log.Info(job.Name)
		}
	}

	// get job controller
	name = "hello-27495875"
	oc, err := jobHandler.GetController(name)
	if err != nil {
		log.Error("get job controller failed")
		log.Error(err)
	} else {
		log.Info("get job controller success")
		log.Info(oc.Name)
		log.Info(oc.UID)
		log.Info(oc.Labels)
	}

	//time.Sleep(time.Second * 5)
	//log.Info("wait job not active")
	//err = jobHandler.WaitNotActive(name)
	//if err != nil {
	//    log.Error(err)
	//}
	//log.Info("job is not active now")
	// is active
	name = "echo"
	for {
		if jobHandler.IsFinish(name) {
			log.Info("job is finished")
			break
		} else {
			log.Info("job is not finised")
		}
		time.Sleep(time.Second)
	}

	// 8. watch job
	log.Info("start watch job")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			jobHandler.Apply(yamlfile)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			//time.Sleep(time.Second * time.Duration(rand.Intn(15)))
			time.Sleep(time.Second * 15)
			jobHandler.Delete(name)
		}
	}()
	jobHandler.Watch(name,
		func(x interface{}) {
			log.Info("added job.")
		},
		func(x interface{}) {
			log.Info("modified job.")
		},
		func(x interface{}) {
			log.Info("deleted job.")
		},
		nil,
	)
}
