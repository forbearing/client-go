package main

import (
	"hybfkuf/pkg/k8s"
	"time"

	log "github.com/sirupsen/logrus"
)

func deploymentExamples() {
	var (
		yamlfile      = "./testData/deployment.yaml"
		name          = "test"
		labelSelector = "type=deployment"
		forceDelete   = false
	)
	deployHandler, err := k8s.NewDeployment(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = deployHandler

	// 1. create deployment
	deployHandler.Delete(name)
	if deploy, err := deployHandler.Create(yamlfile); err != nil {
		log.Error("create deployment failed")
		log.Error(err)
	} else {
		log.Infof("create deployment %q success.", deploy.Name)
	}
	// 2. update deployment
	if deploy, err := deployHandler.Update(yamlfile); err != nil {
		log.Error("update deployment failed")
		log.Error(err)
	} else {
		log.Infof("update deployment %q success.", deploy.Name)
	}
	// 3. apply deployment
	if deploy, err := deployHandler.Apply(yamlfile); err != nil {
		log.Error("apply deployment failed")
		log.Error(err)
	} else {
		log.Infof("apply deployment %q success.", deploy.Name)
	}
	deployHandler.Delete(name)
	// 3. apply deployment
	deployHandler.Delete(name)
	if deploy, err := deployHandler.Apply(yamlfile); err != nil {
		log.Error("apply deployment failed")
		log.Error(err)
	} else {
		log.Infof("apply deployment %q success.", deploy.Name)
	}
	// 4. delete deployment
	if err := deployHandler.Delete(name); err != nil {
		log.Error("delete deployment failed")
		log.Error(err)
	} else {
		log.Infof("delete deployment %q success.", name)
	}
	//4. delete from file
	deployHandler.Apply(yamlfile)
	if err := deployHandler.DeleteFromFile(yamlfile); err != nil {
		log.Error("delete deployment from file failed")
		log.Error(err)
	} else {
		log.Infof("delete deployment %q from file success.", name)
	}
	// 5. get deployment
	deployHandler.Apply(yamlfile)
	time.Sleep(time.Second * 3)
	if deploy, err := deployHandler.Get(name); err != nil {
		log.Error("get deployment failed")
		log.Error(err)
	} else {
		log.Infof("get deployment %q success.", deploy.Name)
	}
	// 6. list deployments by labels
	log.Info()
	if deployList, err := deployHandler.List(labelSelector); err != nil {
		log.Error("list deployments by label failed")
		log.Error(err)
	} else {
		log.Info("list deployments by label success.")
		for _, deploy := range deployList.Items {
			log.Info(deploy.Name)
		}
	}
	// 6. list deployments by namespace
	log.Info()
	if deployList, err := deployHandler.ListByNamespace("kube-system"); err != nil {
		log.Error("list deployments by namespace failed")
		log.Error(err)
	} else {
		log.Info("list deployments by namespace success.")
		for _, deploy := range deployList.Items {
			log.Info(deploy.Name)
		}
	}
	// 6. list all deployments in the k8s cluster
	log.Info()
	if deployList, err := deployHandler.ListAll(); err != nil {
		log.Error("list all deployments in the k8s cluster failed")
		log.Error(err)
	} else {
		log.Info("list all deployments in the k8s cluster success.")
		for _, deploy := range deployList.Items {
			log.Info(deploy.Name)
		}
	}

	// 7. check if the deployment is ready
	log.Info()
	ready := deployHandler.IsReady(name)
	if ready {
		log.Infof("deployment %q is ready.", name)
	} else {
		log.Infof("deployment %q not ready.", name)
		log.Infof("start wait deployment %q to ready.", name)
		// 7. wait for deployment until ready
		err := deployHandler.WaitReady(name, true)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("deployment nginx is ready now.")
	}

	// 8. get deployment details
	log.Info()
	name = "nginx-deploy"
	yamlfile = "./testData/nginx-deploy.yaml"
	deployHandler.Apply(name)
	deployHandler.WaitReady(name, true)
	// get deployment pvc

	log.Info()
	pvcList, err := deployHandler.GetPVC(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("all pvc:")
	for _, pvc := range pvcList {
		log.Info(pvc)
	}

	// get deployment pv
	log.Info()
	pvList, err := deployHandler.GetPV(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("all pv:")
	for _, pv := range pvList {
		log.Info(pv)
	}

	// test WithNamespace
	log.Info()
	log.Info("start test WithNamespace")
	log.Infof("before WithNamespace: %q", deployHandler.Namespace())
	podList, _ := deployHandler.WithNamespace("kube-system").List("k8s-app=kube-dns")
	for _, pod := range podList.Items {
		log.Info(pod.Name)
	}
	log.Infof("after WithNamespace: %q", deployHandler.Namespace())

	podList, _ = deployHandler.List(labelSelector)
	for _, pod := range podList.Items {
		log.Info(pod.Name)
	}
	log.Infof("now WithNamespace: %q", deployHandler.Namespace())

	// test DryRun
	log.Info()
	log.Info("start test WithDryRun")
	name = "test"
	yamlfile = "./testData/deployment.yaml"
	deployHandler.Apply(yamlfile)
	deployHandler.WaitReady(name, true)
	if err := deployHandler.WithDryRun().Delete(name); err != nil {
		log.Error("dry run delete failed")
		log.Error(err)
	} else {
		log.Info("dry run delete success")
	}
	if _, err := deployHandler.WithDryRun().Apply(yamlfile); err != nil {
		log.Error("dry run apply failed")
		log.Error(err)
	} else {
		log.Info("dry run apply success.")
	}

	// 9. watch deployment
	log.Info()
	log.Info("start watch deployment")
	name = "test"
	yamlfile = "./testData/deployment.yaml"
	go func() {
		for {
			deployHandler.Apply(yamlfile)
			time.Sleep(time.Second * 5)
		}
	}()
	go func() {
		for {
			deployHandler.Delete(name)
			time.Sleep(time.Second * 30)
		}
	}()
	deployHandler.Watch(name,
		func(x interface{}) {
			log.Info("added deployHandler.")
		},
		func(x interface{}) {
			log.Info("modified deployHandler.")
		},
		func(x interface{}) {
			log.Info("deleted deployHandler.")
		},
		nil,
	)
}
