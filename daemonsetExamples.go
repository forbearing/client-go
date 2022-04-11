package main

import (
	"hybfkuf/pkg/k8s"

	log "github.com/sirupsen/logrus"
)

func daemonsetExamples() {
	var (
		filepath      = "./examples/daemonset.yaml"
		name          = "test"
		labelSelector = "type=daemonset"
		forceDelete   = false
	)
	dsHandler, err := k8s.NewDaemonSet(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = filepath
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = dsHandler

	// 1. create daemonset
	dsHandler.Delete(name)
	if s, err := dsHandler.Create(filepath); err != nil {
		log.Error("create daemonset failed")
		log.Error(err)
	} else {
		log.Infof("create daemonset %s success.", s.Name)
	}
	// 2. update daemonset
	if s, err := dsHandler.Update(filepath); err != nil {
		log.Error("update daemonset failed")
		log.Error(err)
	} else {
		log.Infof("update daemonset %s success.", s.Name)
	}
	// 3. apply daemonset
	if s, err := dsHandler.Apply(filepath); err != nil {
		log.Error("apply daemonset failed")
		log.Error(err)
	} else {
		log.Infof("apply daemonset %s success.", s.Name)
	}
	dsHandler.Delete(name)
	if s, err := dsHandler.Apply(filepath); err != nil {
		log.Error("apply daemonset failed")
		log.Error(err)
	} else {
		log.Infof("apply daemonset %s success.", s.Name)
	}
	// 4. delete daemonset
	if err := dsHandler.Delete(name); err != nil {
		log.Error("delete daemonset failed")
		log.Error(err)
	} else {
		log.Infof("delete daemonset %s success.", name)
	}
	// delete daemonset from file
	dsHandler.Apply(filepath)
	if err := dsHandler.DeleteFromFile(filepath); err != nil {
		log.Error("delete daemonset from file failed")
		log.Error(err)
	} else {
		log.Infof("delete daemonset %s from file success.", name)
	}
	// 5. get daemonset
	dsHandler.Create(filepath)
	if s, err := dsHandler.Get(name); err != nil {
		log.Error("get daemonset failed")
		log.Error(err)
	} else {
		log.Infof("get daemonset %s success.", s.Name)
	}
	// 6. list daemonset
	if sl, err := dsHandler.List(labelSelector); err != nil {
		log.Error("list daemonset failed")
		log.Error(err)
	} else {
		log.Info("list daemonset success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// wait ready
	log.Info("start wait daemonset ready")
	if err := dsHandler.WaitReady(name, true); err != nil {
		log.Errorf("WaitReady failed.")
		log.Error(err)
	} else {
		log.Info("daemonset is ready now")
	}
	//// 7. watch daemonset
	//log.Info("start watch daemonset")
	//go func() {
	//    for {
	//        rand.Seed(time.Now().UnixNano())
	//        time.Sleep(time.Second * time.Duration(rand.Intn(5)))
	//        dsHandler.Apply(filepath)
	//    }
	//}()
	//go func() {
	//    for {
	//        rand.Seed(time.Now().UnixNano())
	//        time.Sleep(time.Second * 60)
	//        dsHandler.Delete(name)
	//    }
	//}()
	//dsHandler.Watch(name,
	//    func(x interface{}) {
	//        log.Info("add dsHandler.")
	//    },
	//    func(x interface{}) {
	//        log.Info("modified dsHandler.")
	//    },
	//    func(x interface{}) {
	//        log.Info("deleted dsHandler.")
	//    },
	//    nil,
	//)

	dsHandler.Delete(name)
	name = "nginx-ds"
	filepath = "./examples/nginx-ds.yaml"
	dsHandler.Apply(filepath)
	if dsHandler.IsReady(name) {
		log.Infof("daemonset %s is ready.", name)
	} else {
		log.Infof("daemonset %s not ready", name)
		log.Infof("start wait daemonset %s to be ready.", name)
		err = dsHandler.WaitReady(name, true)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("daemonset %s is ready now.", name)
	}

	pvcList, err := dsHandler.GetPVC(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("all pvc:")
	for _, pvc := range pvcList {
		log.Info(pvc)
	}
	pvList, err := dsHandler.GetPV(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("all pv:")
	for _, pv := range pvList {
		log.Info(pv)
	}
	podList, err := dsHandler.GetPods(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("all pod")
	for _, pod := range podList {
		log.Info(pod)
	}

}
