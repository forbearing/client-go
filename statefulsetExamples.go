package main

import (
	"hybfkuf/pkg/k8s"

	log "github.com/sirupsen/logrus"
)

func statefulsetExamples() {
	var (
		yamlfile      = "./testData/statefulset.yaml"
		name          = "test"
		labelSelector = "type=statefulset"
		forceDelete   = false
	)
	stsHandler, err := k8s.NewStatefulSet(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = stsHandler

	// 1. create statefulset
	stsHandler.Delete(name)
	if s, err := stsHandler.Create(yamlfile); err != nil {
		log.Error("create statefulset failed")
		log.Error(err)
	} else {
		log.Infof("create statefulset %s success.", s.Name)
	}
	// 2. update statefulset
	if s, err := stsHandler.Update(yamlfile); err != nil {
		log.Error("update statefulset failed")
		log.Error(err)
	} else {
		log.Infof("update statefulset %s success.", s.Name)
	}
	// 3. apply statefulset
	if s, err := stsHandler.Apply(yamlfile); err != nil {
		log.Error("apply statefulset failed")
		log.Error(err)
	} else {
		log.Infof("apply statefulset %s success.", s.Name)
	}
	stsHandler.Delete(name)
	if s, err := stsHandler.Apply(yamlfile); err != nil {
		log.Error("apply statefulset failed")
		log.Error(err)
	} else {
		log.Infof("apply statefulset %s success.", s.Name)
	}
	// 4. delete statefulset
	if err := stsHandler.Delete(name); err != nil {
		log.Error("delete statefulset failed")
		log.Error(err)
	} else {
		log.Infof("delete statefulset %s success.", name)
	}
	// delete statefulset from file
	stsHandler.Apply(yamlfile)
	if err := stsHandler.DeleteFromFile(yamlfile); err != nil {
		log.Error("delete statefulset from file failed")
		log.Error(err)
	} else {
		log.Infof("delete statefulset %s from file success.", name)
	}
	// 5. get statefulset
	stsHandler.Create(yamlfile)
	if s, err := stsHandler.Get(name); err != nil {
		log.Error("get statefulset failed")
		log.Error(err)
	} else {
		log.Infof("get statefulset %s success.", s.Name)
	}
	// 6. list statefulset
	if sl, err := stsHandler.List(labelSelector); err != nil {
		log.Error("list statefulset failed")
		log.Error(err)
	} else {
		log.Info("list statefulset success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// wait ready
	log.Info("start wait statefulset ready")
	if err := stsHandler.WaitReady(name, true); err != nil {
		log.Errorf("WaitReady failed.")
		log.Error(err)
	} else {
		log.Info("statefulset is ready now")
	}
	//// 7. watch statefulset
	//log.Info("start watch statefulset")
	//go func() {
	//    for {
	//        rand.Seed(time.Now().UnixNano())
	//        time.Sleep(time.Second * time.Duration(rand.Intn(5)))
	//        stsHandler.Apply(yamlfile)
	//    }
	//}()
	//go func() {
	//    for {
	//        rand.Seed(time.Now().UnixNano())
	//        time.Sleep(time.Second * 60)
	//        stsHandler.Delete(name)
	//    }
	//}()
	//stsHandler.Watch(name,
	//    func(x interface{}) {
	//        log.Info("add stsHandler.")
	//    },
	//    func(x interface{}) {
	//        log.Info("modified stsHandler.")
	//    },
	//    func(x interface{}) {
	//        log.Info("deleted stsHandler.")
	//    },
	//    nil,
	//)

	stsHandler.Delete(name)
	name = "nginx-sts"
	yamlfile = "./testData/nginx-sts.yaml"
	stsHandler.Apply(yamlfile)

	if stsHandler.IsReady(name) {
		log.Info("statefulset nginx-sts is ready.")
	} else {
		log.Info("statefulset nginx-sts not ready.")
		log.Info("start to wait statusfulset to be ready.")
		stsHandler.WaitReady(name, true)
		log.Info("statefulset nginx-sts is ready now.")
	}

	pvcList, err := stsHandler.GetPVC(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("all pvc")
	for _, pvc := range pvcList {
		log.Info(pvc)
	}
	pvList, err := stsHandler.GetPV(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("all pv")
	for _, pv := range pvList {
		log.Info(pv)
	}
	podList, err := stsHandler.GetPods(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("all pods")
	for _, pod := range podList {
		log.Info(pod)
	}
}
