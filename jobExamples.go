package main

import (
	"hybfkuf/pkg/k8s"
	"time"

	log "github.com/sirupsen/logrus"
)

func jobExamples() {
	var (
		filepath      = "./examples/job.yaml"
		name          = "echo"
		labelSelector = "job-name=echo"
		forceDelete   = false
	)
	job, err := k8s.NewJob(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = filepath
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = job

	// 1. create job
	job.Delete(name)
	time.Sleep(time.Second * 3)
	if s, err := job.Create(filepath); err != nil {
		log.Error("create job failed")
		log.Error(err)
	} else {
		log.Infof("create job %s success.", s.Name)
	}
	//// 2. update job
	//if s, err := job.Update(filepath); err != nil {
	//    log.Error(err)
	//} else {
	//    log.Infof("update job %s success.", s.Name)
	//}
	//// 3. apply job
	//if s, err := job.Apply(filepath); err != nil {
	//    log.Error(err)
	//} else {
	//    log.Infof("apply job %s success.", s.Name)
	//}
	//job.Delete(name, forceDelete)
	//if s, err := job.Apply(filepath); err != nil {
	//    log.Error(err)
	//} else {
	//    log.Infof("apply job %s success.", s.Name)
	//}
	// 4. delete job
	if err := job.Delete(name); err != nil {
		log.Error("delete job failed")
		log.Error(err)
	} else {
		log.Infof("delete job %s success.", name)
	}
	time.Sleep(time.Second * 3)
	// 5. get job
	job.Create(filepath)
	if s, err := job.Get(name); err != nil {
		log.Error("get job failed")
		log.Error(err)
	} else {
		log.Infof("get job %s success.", s.Name)
	}
	// 6. list job
	if sl, err := job.List(labelSelector); err != nil {
		log.Error("list job failed")
		log.Error(err)
	} else {
		log.Info("list job success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}

	//// wait not active
	//time.Sleep(time.Second * 5)
	//log.Info("wait job not active")
	//err = job.WaitNotActive(name)
	//if err != nil {
	//    log.Error(err)
	//}
	//log.Info("job is not active now")
	// is active
	for {
		if job.IsFinish(name) {
			log.Info("job is finished")
		} else {
			log.Info("job is not finised")
		}
		time.Sleep(time.Second)
	}
	//// 7. watch job
	//log.Info("start watch job")
	//go func() {
	//    for {
	//        rand.Seed(time.Now().UnixNano())
	//        time.Sleep(time.Second * time.Duration(rand.Intn(5)))
	//        job.Apply(filepath)
	//    }
	//}()
	//go func() {
	//    for {
	//        rand.Seed(time.Now().UnixNano())
	//        //time.Sleep(time.Second * time.Duration(rand.Intn(15)))
	//        time.Sleep(time.Second * 15)
	//        job.Delete(name, forceDelete)
	//    }
	//}()
	//job.Watch(name,
	//    func() {
	//        log.Info("add job.")
	//    },
	//    func() {
	//        log.Info("modified job.")
	//    },
	//    func() {
	//        log.Info("deleted job.")
	//    },
	//)
}
