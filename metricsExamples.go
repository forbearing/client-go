package main

import (
	"context"
	"hybfkuf/pkg/k8s"

	log "github.com/sirupsen/logrus"
)

func metricsExmaples() {
	var (
		name            string
		label           string
		namespace       string
		ctx             = context.TODO()
		nodeMetricsList []k8s.NodeMetrics
		nodeMetrics     *k8s.NodeMetrics
		podMetricsList  []k8s.PodMetrics
		podMetrics      *k8s.PodMetrics
		err             error
	)
	_ = name
	_ = label
	_ = namespace
	_ = nodeMetricsList
	_ = nodeMetrics
	_ = podMetricsList
	_ = podMetrics
	_ = err

	metricsHandler, err := k8s.NewMetrics(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	// query pod metrics by name
	name = "nginx-pod"
	podMetrics, err = metricsHandler.Pod(name)
	if err != nil {
		log.Error("query pod metrics error")
		log.Error(err)
	} else {
		log.Info("query pod metrics success.")
	}
	//log.Info(podMetrics.Timestamp)
	//log.Info(podMetrics.Window)
	log.Info(podMetrics.ObjectMeta.Name, podMetrics.Containers)

	// query pod metrics by label
	label = "k8s-app=metrics-server"
	namespace = "kube-system"
	log.Info("namespace: ", metricsHandler.Namespace())
	podMetricsList, err = metricsHandler.WithNamespace(namespace).Pods(label)
	for _, podMetrics := range podMetricsList {
		//log.Info(podMetrics.Timestamp)
		//log.Info(podMetrics.Window)
		log.Info(podMetrics.ObjectMeta.Name, podMetrics.Containers)
	}
	log.Info("namespace: ", metricsHandler.Namespace())

	// query node metrics by name
	log.Info()
	name = "d11-k8s-master1"
	nodeMetrics, err = metricsHandler.Node(name)
	if err != nil {
		log.Error("query node metrics failed")
		log.Error(err)
	} else {
		log.Info("query node metrics success.")
		//log.Info(nodeMetrics.Timestamp)
		//log.Info(nodeMetrics.Window)
		log.Info(nodeMetrics.ObjectMeta.Name)
		log.Info(nodeMetrics.Usage)
	}

	// query node metrics by label
	log.Info()
	label = `node-role.kubernetes.io/master`
	nodeMetricsList, err = metricsHandler.Nodes(label)
	if err != nil {
		log.Error("query node metrics failed")
		log.Error(err)
	} else {
		log.Info("query node metrics success.")
		for _, nodeMetrics := range nodeMetricsList {
			//log.Info(nodeMetrics.Timestamp)
			//log.Info(nodeMetrics.Window)
			log.Info(nodeMetrics.ObjectMeta.Name)
			log.Info(nodeMetrics.Usage)
		}
	}
	label = "!node-role.kubernetes.io/master"
	nodeMetricsList, err = metricsHandler.Nodes(label)
	if err != nil {
		log.Error("query node metrics failed")
		log.Error(err)
	} else {
		log.Info("query node metrics success.")
		for _, nodeMetrics := range nodeMetricsList {
			//log.Info(nodeMetrics.Timestamp)
			//log.Info(nodeMetrics.Window)
			log.Info(nodeMetrics.ObjectMeta.Name)
			log.Info(nodeMetrics.Usage)
		}
	}

	// query node metrics by name using REST API
	log.Info()
	name = "d11-k8s-master1"
	nodeMetrics, err = metricsHandler.NodeRaw(name)
	if err != nil {
		log.Error("query node metrics by name using REST API failed")
		log.Error(err)
	} else {
		log.Info("query node metrics by name using REST API success.")
		log.Info(nodeMetrics.ObjectMeta.Name)
		log.Info(nodeMetrics.Usage)
	}
	// query all node metrics using REST API
	log.Info()
	nodeMetricsList, err = metricsHandler.NodeAllRaw()
	if err != nil {
		log.Error("query all node metrics using REST API failed")
		log.Error(err)
	} else {
		log.Info("query all node metrics using REST API success.")
		for _, nodeMetrics := range nodeMetricsList {
			log.Info(nodeMetrics.ObjectMeta.Name)
			log.Info(nodeMetrics.Usage)
		}
	}

	/// query pod metrics by name using REST API
	log.Info()
	name = "nginx-pod"
	podMetrics, err = metricsHandler.PodRaw(name)
	if err != nil {
		log.Error("query pod metrics by name using REST API failed")
		log.Error(err)
	} else {
		log.Info("query pod metrics by name using REST API success.")
		log.Info(podMetrics.ObjectMeta.Name, podMetrics.Containers)
	}
	// query all the pod metrics in the namespace where pod is runing, using REST API
	log.Info()
	podMetricsList, err = metricsHandler.WithNamespace("kube-system").PodsRaw()
	if err != nil {
		log.Error("query all the pod metrics in the namespace where the pod is running failed")
		log.Error(err)
	} else {
		log.Info("query all the pod metrics in the namespace where the pod is running success.")
		for _, podMetrics := range podMetricsList {
			log.Info(podMetrics.ObjectMeta.Name, podMetrics.Containers)
		}
	}
	// query all the pod metrics in the k8s cluster  where pod is runing, using REST API
	log.Info()
	podMetricsList, err = metricsHandler.WithNamespace("kube-system").PodAllRaw()
	if err != nil {
		log.Error("query all the pod metrics in the k8s cluster where the pod is running failed")
		log.Error(err)
	} else {
		log.Info("query all the pod metrics in the k8s cluster where the pod is running success.")
		for _, podMetrics := range podMetricsList {
			log.Info(podMetrics.ObjectMeta.Name, podMetrics.Containers)
		}
	}
}
