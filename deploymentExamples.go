package main

import (
	"hybfkuf/pkg/k8s"
	"time"

	log "github.com/sirupsen/logrus"
)

func deploymentExamples() {
	var (
		filepath      = "./examples/deployment.yaml"
		name          = "test"
		labelSelector = "type=deployment"
		forceDelete   = false
	)
	deployHandler, err := k8s.NewDeployment(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = filepath
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = deployHandler

	log.Info(deployHandler.Namespace())
	deployHandler.WithNamespace("kube-system").List("k8s-ap=kube-dns")
	log.Info(deployHandler.Namespace())

	// 1. create deployment
	deployHandler.Delete(name)
	if s, err := deployHandler.Create(filepath); err != nil {
		log.Error("create deployment failed")
		log.Error(err)
	} else {
		log.Infof("create deployment %s success.", s.Name)
	}
	// 2. update deployment
	if s, err := deployHandler.Update(filepath); err != nil {
		log.Error("update deployment failed")
		log.Error(err)
	} else {
		log.Infof("update deployment %s success.", s.Name)
	}
	// 3. apply deployment
	if s, err := deployHandler.Apply(filepath); err != nil {
		log.Error("apply deployment failed")
		log.Error(err)
	} else {
		log.Infof("apply deployment %s success.", s.Name)
	}
	//{
	//    // 3. apply deployment
	//    var err error
	//    var deploy *appsv1.Deployment
	//    deploy, err = deployHandler.Apply(filepath)
	//    if err != nil {
	//        log.Error("apply deployment failed")
	//        log.Error(err)
	//    } else {
	//        log.Info(err)
	//        log.Infof("apply deployment %s success.", deploy.Name)
	//    }
	//    select {}
	//}
	deployHandler.Delete(name)
	if s, err := deployHandler.Apply(filepath); err != nil {
		log.Error("apply deployment failed")
		log.Error(err)
	} else {
		log.Infof("apply deployment %s success.", s.Name)
	}
	// 4. delete deployment
	if err := deployHandler.Delete(name); err != nil {
		log.Error("delete deployment failed")
		log.Error(err)
	} else {
		log.Infof("delete deployment %s success.", name)
	}
	// 5. delete from file
	deployHandler.Apply(filepath)
	if err := deployHandler.DeleteFromFile(filepath); err != nil {
		log.Error("delete deployment from file failed")
		log.Error(err)
	} else {
		log.Info("delete deployment from file success.")
	}
	// 6. get deployment
	deployHandler.Apply(filepath)
	time.Sleep(time.Second * 3)
	if s, err := deployHandler.Get(name); err != nil {
		log.Error("get deployment failed")
		log.Error(err)
	} else {
		log.Infof("get deployment %s success.", s.Name)
	}
	// 7. list deployment
	if sl, err := deployHandler.List(labelSelector); err != nil {
		log.Error("list deployment failed")
		log.Error(err)
	} else {
		log.Info("list deployment success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// wait ready
	log.Info("start wait deployment ready")
	log.Info(deployHandler.Namespace())
	if err := deployHandler.WaitReady(name, true); err != nil {
		log.Errorf("WaitReady failed.")
		log.Error(err)
	} else {
		log.Info("deployment is ready now")
	}

	//// 8. watch deployment
	//log.Info("start watch deployment")
	//deployHandler.Watch(name,
	//    func(x interface{}) {
	//        log.Info("add deployHandler.")
	//    },
	//    func(x interface{}) {
	//        log.Info("modified deployHandler.")
	//    },
	//    func(x interface{}) {
	//        log.Info("deleted deployHandler.")
	//    },
	//    nil,
	//)

	deployHandler.Delete(name)
	name = "nginx-deploy"
	filepath = "./examples/nginx-deploy.yaml"
	deployHandler.Apply(name)
	// 判断 deployment 是否 ready
	ready := deployHandler.IsReady(name)
	if ready {
		log.Info("deployment nginx is ready.")
	} else {
		log.Info("deployment nginx not ready.")
		log.Info("start wait deploy to ready")
		err := deployHandler.WaitReady(name, true)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("deployment nginx is ready now.")
	}

	// 获取 deployment 中所有的 pvc
	pvcList, err := deployHandler.GetPVC(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("all pvc:")
	for _, pvc := range pvcList {
		log.Info(pvc)
	}

	// 获取 deployment 中所有的 pv
	pvList, err := deployHandler.GetPV(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("all pv:")
	for _, pv := range pvList {
		log.Info(pv)
	}

	log.Info()
	log.Info("test WithNamespace")
	log.Info(deployHandler.Namespace())
	podList, _ := deployHandler.WithNamespace("kube-system").List("k8s-app=kube-dns")
	for _, pod := range podList.Items {
		log.Info(pod.Name)
	}
	podList, _ = deployHandler.List(labelSelector)
	for _, pod := range podList.Items {
		log.Info(pod.Name)
	}
	log.Info(deployHandler.Namespace())

	log.Info()
	name = "test"
	filepath = "./examples/deployment.yaml"
	log.Info("test WithDryRun")
	deployHandler.Apply(filepath)
	log.Infof("start wait %s ready", name)
	deployHandler.WaitReady(name, true)
	if err := deployHandler.WithDryRun().Delete(name); err != nil {
		log.Error("dry run delete failed")
		log.Error(err)
	} else {
		log.Info("dry run delete success")
	}
	if _, err := deployHandler.WithDryRun().Apply(filepath); err != nil {
		log.Error("dry run apply failed")
		log.Error(err)
	} else {
		log.Info("dry run apply success.")
	}
	deployHandler.Delete(name)
}
