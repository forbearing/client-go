package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func podExamples() {
	var (
		yamlfile      = "./testData/pod.yaml"
		name          = "test"
		labelSelector = "type=pod"
	)
	podHandler, err := k8s.NewPod(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = podHandler

	// 1. create pod
	podHandler.Delete(name)
	time.Sleep(time.Second * 5)
	if pod, err := podHandler.Create(yamlfile); err != nil {
		log.Error("create pod failed")
		log.Error(err)
	} else {
		log.Infof("create pod %q success.", pod.Name)
	}
	//// 2. update pod
	//if pod, err := podHandler.Update(yamlfile); err != nil {
	//    log.Error(err)
	//} else {
	//    log.Infof("update pod %q success.", pod.Name)
	//}
	//// 3. apply pod
	//if pod, err := podHandler.Apply(yamlfile); err != nil {
	//    log.Error(err)
	//} else {
	//    log.Infof("apply pod %q success.", pod.Name)
	//}
	//podHandler.Delete(name)
	//if pod, err := podHandler.Apply(yamlfile); err != nil {
	//    log.Error(err)
	//} else {
	//    log.Infof("apply pod %q success.", pod.Name)
	//}
	// 4. delete pod
	if err := podHandler.Delete(name); err != nil {
		log.Error("delete pod failed")
		log.Error(err)
	} else {
		log.Infof("delete pod %q success.", name)
	}
	time.Sleep(time.Second * 5)
	podHandler.Create(yamlfile)
	time.Sleep(time.Second * 2)
	// delete pod from file
	if err := podHandler.DeleteFromFile(yamlfile); err != nil {
		log.Error("delete pod from file failed")
		log.Error(err)
	} else {
		log.Infof("delete pod %q from file success.", name)
	}
	// 5. get pod
	podHandler.Create(yamlfile)
	if pod, err := podHandler.Get(name); err != nil {
		log.Error("get pod failed")
		log.Error(err)
	} else {
		log.Infof("get pod %q success.", pod.Name)
	}
	// 6. list pod by label
	if sl, err := podHandler.List(labelSelector); err != nil {
		log.Error("list pod failed")
		log.Error(err)
	} else {
		log.Info("list pod success.")
		for _, pod := range sl.Items {
			log.Info(pod.Name)
		}
	}
	//  list all pods in namespace where the pod is running
	log.Info()
	if podList, err := podHandler.ListByNamespace(NAMESPACE); err != nil {
		log.Error("list pod by namespace failed")
		log.Error(err)
	} else {
		log.Info("list pod by namespace success.")
		for _, pod := range podList.Items {
			log.Info(pod.Name)
		}
	}
	//  list al pods in k8s node where the pod is running
	log.Info()
	if podList, err := podHandler.ListByNode("d11-k8s-master1"); err != nil {
		log.Error("list all pods by k8s node failed")
		log.Error(err)
	} else {
		log.Info("list all pods by k8s node success.")
		for _, pod := range podList.Items {
			log.Info(pod.Name)
		}
	}

	//  list all pod in k8s cluster where the pod is running
	log.Info()
	if podList, err := podHandler.ListAll(); err != nil {
		log.Error("list all pods failed")
		log.Error(err)
	} else {
		log.Info("list all pods success.")
		for _, pod := range podList.Items {
			log.Info(pod.Name)
		}
	}

	// 7 get pod details
	log.Info()
	name = "nginx-pod"
	labelSelector = "app=nginx-pod"
	yamlfile = "./testData/nginx-pod.yaml"
	podHandler.Apply(yamlfile)
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
		log.Infof("pod ip: %q", ip)
	}
	// get pod uid
	uid, err := podHandler.GetUID(name)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Infof("pod uid: %q", uid)
	}
	// get pod node ip
	nodeIP, err := podHandler.GetNodeIP(name)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Infof("node ip: %q", nodeIP)
	}
	// get pod node name
	nodeName, err := podHandler.GetNodeName(name)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Infof("node name: %q", nodeName)
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
	yamlfile = "./testData/pod.yaml"
	go func() {
		for {
			_, err := podHandler.Apply(yamlfile)
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
	//if _, err := podHandler.WithDryRun().Create(yamlfile); err != nil {
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
	oc, err := podHandler.GetController(name)
	if err != nil {
		log.Error("get owner references failed")
		log.Error(err)
	} else {
		log.Info("get owner references success.")
		log.Info(oc.Kind)
		log.Info(oc.Name)
		log.Info(oc.Labels)
		log.Info(oc.Ready)
		log.Info(oc.Images)
		log.Info(oc.CreationTimestamp)
	}

	// 8. watch pod
	log.Info("start watch pod")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			podHandler.Apply(yamlfile)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(10)))
			//time.Sleep(time.Second * 125)
			podHandler.Delete(name)
		}
	}()
	podHandler.Watch(name,
		func(x interface{}) {
			log.Info("added podHandler.")
		},
		func(x interface{}) {
			log.Info("modified podHandler.")
		},
		func(x interface{}) {
			log.Info("deleted podHandler.")
		},
		nil,
	)
}
