package main

import (
	"context"
	"flag"
	"path/filepath"

	//"k8s/pkg/k8s"
	//k8sdynamic "k8s/pkg/k8s/dynamic"

	"k8s.io/client-go/util/homedir"
)

var (
	ctx        = context.Background()
	NAMESPACE  = "test"
	kubeconfig *string
)

func init() {
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
}

func main() {
	//clusterrolebindingExamples()
	//clusterroleExamples()
	//configmapExamples()
	//cronjobExamples()
	//metricsExmaples()
	//applyExamples()
	//deploymentExamples()
	podExamples()
	//statefulsetExamples()
	//daemonsetExamples()
	//nodeExamples()
	//namespaceExamples()
	//serviceExamples()
	//secretExamples()
	//serviceaccountExamples()
	//persistentvolumeExamples()
	//persistentvolumeclaimExamples()
	//jobExamples()
	//cronjobExamples()
	//ingressExamples()
	//ingressclassExamples()
	//roleExamples()
	//rolebindingExamples()
	//networkpolicyExamples()
	//applyFile()
	//kubectl()
	//testDeploy()
	//testPod()
	//testStatefulset()
	//testDaemonset()
}

//func dynamicPodsExamples() {
//    var (<<<
//        client        dynamic.Interface
//        name          = "pod-nginx"
//        filepath      = "./examples/pod-nginx.yaml"
//        labelSelector = "app=k8s-tools"
//        forceDelete   = false
//        err           error
//    )
//    _ = client
//    _ = name
//    _ = filepath
//    _ = labelSelector
//    _ = forceDelete
//    _ = err

//    // create dynamic client
//    client, err = k8sdynamic.New()
//    if err != nil {
//        log.Fatal()
//    }

//    // 1. create pod
//    if pod, err := k8sdynamic.CreatePods(ctx, client, NAMESPACE, filepath); err != nil {
//        log.Error(err)
//    } else {
//        log.Info("create pod success.")
//        log.Info(pod.Name)
//    }

//    //// 2. Update pod
//    //// pod 几乎所有字段都是不可变的, 更新 pod 往往会失败, 所有不演示 update pod
//    //pod, err = k8sdynamic.UpdatePods(ctx, client, NAMESPACE, filepath)
//    //if err != nil {
//    //    log.Error(err)
//    //} else {
//    //    log.Info("update pod success.")
//    //    log.Info(pod.Name)
//    //}

//    // 3. delete pod
//    if err := k8sdynamic.DeletePod(ctx, client, NAMESPACE, name, forceDelete); err != nil {
//        log.Error(err)
//    } else {
//        log.Info("delete pod success.")
//    }
//    if _, err := k8sdynamic.CreatePods(ctx, client, NAMESPACE, filepath); err != nil {
//        log.Error()
//    }

//    // 4. Get pod
//    if pod, err := k8sdynamic.GetPod(ctx, client, NAMESPACE, name); err != nil {
//        log.Error(err)
//    } else {
//        log.Info("get pod success.")
//        log.Info(pod.Name)
//    }

//    // 5. list pods
//    if podList, err := k8sdynamic.ListPods(ctx, client, NAMESPACE, labelSelector); err != nil {
//        log.Error(err)
//    } else {
//        log.Info("list pods success.")
//        for _, pod := range podList.Items {
//            log.Info(pod.Name)
//        }
//    }

//    // 6. watch pods
//    k8sdynamic.WatchPods(ctx, client, NAMESPACE, labelSelector)
//}>>>
//func dynamicDeploymentsExamples() {
//    var (<<<
//        client        dynamic.Interface
//        err           error
//        name          = "k8s-tools"
//        filepath      = "./examples/k8s-tools.yaml"
//        labelSelector = "app=k8s-tools"
//        forceDelete   = false
//    )
//    _ = client
//    _ = name
//    _ = labelSelector
//    _ = filepath
//    _ = forceDelete

//    // create dynamic client
//    if client, err = k8sdynamic.New(); err != nil {
//        log.Fatal()
//    }

//    // 1. create deployment
//    if deploy, err := k8sdynamic.CreateDeployment(ctx, client, NAMESPACE, filepath); err != nil {
//        log.Error(err)
//    } else {
//        log.Info("create deployment success.")
//        log.Info(deploy.Name)
//    }

//    // 2. update deployment
//    if deploy, err := k8sdynamic.UpdateDeployment(ctx, client, NAMESPACE, filepath); err != nil {
//        log.Error(err)
//    } else {
//        log.Info("update deployment success.")
//        log.Info(deploy.Name)
//    }

//    // 3. delete deployment
//    if err = k8sdynamic.DeleteDeployment(ctx, client, NAMESPACE, name, forceDelete); err != nil {
//        log.Error(err)
//    } else {
//        log.Info("delete deployment k8s-tools success")
//    }
//    if _, err = k8sdynamic.CreateDeployment(ctx, client, NAMESPACE, filepath); err != nil {
//        log.Error(err)
//    }

//    // 4. get deployment
//    if deploy, err := k8sdynamic.GetDeployment(ctx, client, NAMESPACE, name); err != nil {
//        log.Error(err)
//    } else {
//        log.Info("get deployment success.")
//        log.Info(deploy.GetName())
//        log.Info(*deploy.Spec.Replicas)
//    }

//    // 5. list deployment
//    if deployList, err := k8sdynamic.ListDepoyments(ctx, client, NAMESPACE, labelSelector); err != nil {
//        log.Error(err)
//    } else {
//        log.Info("list deployments success.")
//        for _, deploy := range deployList.Items {
//            log.Info(deploy.Name)
//        }
//    }

//    // 6. watch deployment
//    k8sdynamic.WatchDeployments(ctx, client, NAMESPACE, labelSelector)
//}>>>
