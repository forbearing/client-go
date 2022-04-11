package main

import (
	"hybfkuf/pkg/k8s"
	"time"

	log "github.com/sirupsen/logrus"
)

func podExamples() {
	var (
		filepath      = "./examples/pod.yaml"
		name          = "test"
		labelSelector = "type=pod"
		forceDelete   = false
	)
	podHandler, err := k8s.NewPod(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = filepath
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = podHandler

	// 1. create pod
	podHandler.Delete(name)
	time.Sleep(time.Second * 5)
	if s, err := podHandler.Create(filepath); err != nil {
		log.Error("create pod failed")
		log.Error(err)
	} else {
		log.Infof("create pod %s success.", s.Name)
	}
	//// 2. update pod
	//if s, err := podHandler.Update(filepath); err != nil {
	//    log.Error(err)
	//} else {
	//    log.Infof("update pod %s success.", s.Name)
	//}
	//// 3. apply pod
	//if s, err := podHandler.Apply(filepath); err != nil {
	//    log.Error(err)
	//} else {
	//    log.Infof("apply pod %s success.", s.Name)
	//}
	//podHandler.Delete(name)
	//if s, err := podHandler.Apply(filepath); err != nil {
	//    log.Error(err)
	//} else {
	//    log.Infof("apply pod %s success.", s.Name)
	//}
	// 4. delete pod
	if err := podHandler.Delete(name); err != nil {
		log.Error("delete pod failed")
		log.Error(err)
	} else {
		log.Infof("delete pod %s success.", name)
	}
	time.Sleep(time.Second * 5)
	podHandler.Create(filepath)
	time.Sleep(time.Second * 2)
	// delete pod from file
	if err := podHandler.DeleteFromFile(filepath); err != nil {
		log.Error("delete pod from file failed")
		log.Error(err)
	} else {
		log.Infof("delete pod %s from file success.", name)
	}
	// 5. get pod
	podHandler.Create(filepath)
	if s, err := podHandler.Get(name); err != nil {
		log.Error("get pod failed")
		log.Error(err)
	} else {
		log.Infof("get pod %s success.", s.Name)
	}
	// 6. list pod
	if sl, err := podHandler.List(labelSelector); err != nil {
		log.Error("list pod failed")
		log.Error(err)
	} else {
		log.Info("list pod success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	//// 7. watch pod
	//log.Info("start watch pod")
	//go func() {
	//    for {
	//        rand.Seed(time.Now().UnixNano())
	//        time.Sleep(time.Second * time.Duration(rand.Intn(5)))
	//        podHandler.Apply(filepath)
	//    }
	//}()
	//go func() {
	//    for {
	//        rand.Seed(time.Now().UnixNano())
	//        time.Sleep(time.Second * time.Duration(rand.Intn(10)))
	//        //time.Sleep(time.Second * 125)
	//        podHandler.Delete(name)
	//    }
	//}()
	//podHandler.Watch(name,
	//    func(x interface{}) {
	//        log.Info("add podHandler.")
	//    },
	//    func(x interface{}) {
	//        log.Info("modified podHandler.")
	//    },
	//    func(x interface{}) {
	//        log.Info("deleted podHandler.")
	//    },
	//    nil,
	//)

	podHandler.Delete(name)
	name = "nginx-pod"
	labelSelector = "app=nginx-pod"
	filepath = "./examples/nginx-pod.yaml"
	podHandler.Apply(filepath)
	// wait ready
	ready := podHandler.IsReady(name)
	if ready {
		log.Info("pod nginx is ready")
	} else {
		log.Info("pod nginx not ready")
		log.Info("start wait pod nginx to be ready.")
		podHandler.WaitReady("nginx-pod", true)
		log.Info("pod nginx is ready now.")
	}

	// get pod pvc
	pvcList, err := podHandler.GetPVC(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("all pvc:")
	for _, pvc := range pvcList {
		log.Info(pvc)
	}
	// get pod pv
	pvList, err := podHandler.GetPV(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("all pv:")
	for _, pv := range pvList {
		log.Info(pv)
	}
	// get pod ip
	ip, err := podHandler.GetIP(name)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Infof("pod ip: %s", ip)
	}
	// get pod uid
	uid, err := podHandler.GetUID(name)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Infof("pod uid: %s", uid)
	}
	// get pod node ip
	nodeIP, err := podHandler.GetNodeIP(name)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Infof("node ip: %s", nodeIP)
	}
	// get pod node name
	nodeName, err := podHandler.GetNodeName(name)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Infof("node name: %s", nodeName)
	}
	// get pod containers
	containerList, err := podHandler.GetContainers(name)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Info("all containers:")
		for _, container := range containerList {
			log.Info(container)
		}
	}
	// test WithNamespace
	log.Info("test WithNamespace")
	podList, _ := podHandler.WithNamespace("kube-system").List("k8s-app=kube-dns")
	for _, pod := range podList.Items {
		log.Info(pod.Name)
	}
	podList, _ = podHandler.List(labelSelector)
	for _, pod := range podList.Items {
		log.Info(pod.Name)
	}

	// test WithDryRun
	log.Info("test WithDryRun")
	name = "test"
	filepath = "./examples/pod.yaml"
	go func() {
		for {
			_, err := podHandler.Apply(filepath)
			if err == nil {
				return
			}
			time.Sleep(time.Second * 1)
		}
	}()
	log.Infof("start wait %q ready", name)
	podHandler.WaitReady(name, false)
	if err := podHandler.WithDryRun().Delete(name); err != nil {
		log.Error("dry run delete failed.")
		log.Error(err)
	} else {
		log.Info("dry run delete success.")
	}
	//time.Sleep(time.Second * 5)
	//podHandler.Delete(name)
	//time.Sleep(time.Second * 5)
	//if _, err := podHandler.WithDryRun().Create(filepath); err != nil {
	//    log.Error("dry run apply failed")
	//    log.Error(err)
	//} else {
	//    log.Info("dry run apply success.")
	//}

	// execute command in pod
	name = "nginx-pod"
	podHandler.WaitReady(name, true)
	log.Info("execute command in pod")
	//cmd := []string{
	//    "sh",
	//    "-c",
	//    "apt update",
	//}
	cmd := []string{
		"sh",
		"-c",
		"cat /etc/os-release",
	}
	err = podHandler.Execute(name, "nginx", cmd)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("execute command success.")

	// get owner references
	name = "nginx-sts-0"
	ocs, err := podHandler.GetOwnerController(name)
	if err != nil {
		log.Error("get owner references failed")
		log.Error(err)
	} else {
		log.Info("get owner references success.")
		for _, oc := range ocs {
			log.Info(oc.Kind)
			log.Info(oc.Name)
			log.Info(oc.Labels)
			log.Info(oc.Ready)
			log.Info(oc.Images)
			log.Info(oc.CreationTimestamp)
		}
	}
}
