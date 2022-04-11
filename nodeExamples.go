package main

import (
	"context"
	"hybfkuf/pkg/k8s"

	log "github.com/sirupsen/logrus"
)

func nodeExamples() {
	var nodeName string
	nodeHandler, err := k8s.NewNode(context.TODO(), *kubeconfig)
	if err != nil {
		log.Error("new a nodeHandler error")
		log.Error(err)
		return
	}

	// 1. list node
	label := "node-role.kubernetes.io/master"
	nodeList, err := nodeHandler.List(label)
	if err != nil {
		log.Error("list node failed")
		log.Error(err)
	} else {
		log.Info("list node success")
		for _, node := range nodeList.Items {
			log.Info(node.Name)
		}
	}
	// 2. get node
	nodeName = "d11-k8s-worker3"
	node, err := nodeHandler.Get(nodeName)
	if err != nil {
		log.Error("get node failed")
		log.Error(err)
	} else {
		log.Info("get node success.")
		log.Info(node.Spec)
	}
	// check if the node is ready
	if nodeHandler.IsReady(nodeName) {
		log.Infof("%s is ready", nodeName)
	} else {
		log.Infof("%s not ready", nodeName)
	}

	// get the node status
	nodeStatus := nodeHandler.GetStatus(nodeName)
	log.Info("message: ", nodeStatus.Message)
	log.Info("Reason:  ", nodeStatus.Reason)
	log.Info("Status:  ", nodeStatus.Status)

	// get the node roles
	//nodeName = "d11-k8s-worker4"
	nodeName = "d11-k8s-master3"
	roles := nodeHandler.GetRoles(nodeName)
	log.Infof("the roles of the node %s: is %v", nodeName, roles)

	//// check if the node is master, control-plane
	////nodeName = "d11-k8s-master1"
	//nodeName = "d11-k8s-worker1"
	//if nodeHandler.IsMaster(nodeName) {
	//    log.Infof("%s is master", nodeName)
	//} else {
	//    log.Infof("%s is not master", nodeName)
	//}
	//if nodeHandler.IsControlPlane(nodeName) {
	//    log.Infof("%s is control-plane", nodeName)
	//} else {
	//    log.Infof("%s is not control-plane", nodeName)
	//}

	//// get all pods in the node
	//nodeName = "d11-k8s-worker1"
	//podList, err := nodeHandler.GetPods(nodeName)
	//if err != nil {
	//    log.Error("get pods error")
	//    log.Info(err)
	//} else {
	//    log.Info()
	//    log.Info("get pods success.")
	//    for _, pod := range podList.Items {
	//        log.Println(pod.Name)
	//    }
	//    log.Info()
	//}

	// get all  non terminated pods in the node
	nodeName = "d11-k8s-worker1"
	podList2, err := nodeHandler.GetNonTerminatedPods(nodeName)
	if err != nil {
		log.Error("get non terminated pods error")
		log.Error(err)
	} else {
		log.Info()
		log.Info("get non terminated pods success.")
		for _, pod := range podList2.Items {
			log.Println(pod.Name)
		}
		log.Info()
	}

	//// get master node info
	//masterInfo, err := nodeHandler.GetMasterInfo()
	//if err != nil {
	//    log.Error("get master node info failed")
	//    log.Error(err)
	//} else {
	//    log.Info("get master node info success.")
	//    for _, info := range masterInfo {
	//        log.Info(info)
	//    }
	//}

	//// get worker node info
	//workerInfo, err := nodeHandler.GetWorkerInfo()
	//if err != nil {
	//    log.Error("get worker node info failed")
	//    log.Error(err)
	//} else {
	//    log.Info("get worker node info success.")
	//    for _, info := range workerInfo {
	//        log.Info(info)
	//    }
	//}
	//// get all k8s node info
	//workerInfo, err := nodeHandler.GetAllInfo()
	//if err != nil {
	//    log.Error("get all k8s node info failed")
	//    log.Error(err)
	//} else {
	//    log.Info("get all k8s node info success.")
	//    for _, info := range workerInfo {
	//        log.Info(info)
	//    }
	//}

	// get node ip
	nodeName = "d11-k8s-master1"
	ip, err := nodeHandler.GetIP(nodeName)
	if err != nil {
		log.Error("get node ip error")
		log.Error(err)
	} else {
		log.Info("get node ip success.")
		log.Info(ip)
	}
	// get node hostname
	nodeName = "d11-k8s-master1"
	hostname, err := nodeHandler.GetHostname(nodeName)
	if err != nil {
		log.Error("get node hostname error")
		log.Error(err)
	} else {
		log.Info("get node hostname success.")
		log.Info(hostname)
	}

	// get node cidr
	nodeName = "d11-k8s-master1"
	cidr, _ := nodeHandler.GetCIDR(nodeName)
	log.Info(cidr)
	// get node cidrs
	cidrs, _ := nodeHandler.GetCIDRs(nodeName)
	log.Info(cidrs)

}
