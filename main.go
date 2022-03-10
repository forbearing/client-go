package main

import (
	"context" // {{{
	"flag"
	"path/filepath"

	//"hybfkuf/pkg/k8s"
	k8sclient "hybfkuf/pkg/k8s/clientset"
	//k8sdynamic "hybfkuf/pkg/k8s/dynamic"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/homedir" // }}}
)

var (
	ctx        = context.Background() // {{{
	namespace  = "test"
	kubeconfig *string // }}}
)

func init() {
	if home := homedir.HomeDir(); home != "" { // {{{
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse() // }}}
}

func main() {
	//dynamicDeploymentsExamples()
	//dynamicPodsExamples()
	//deploymentExamples()
	//deploymentWatcher()
	//nodeExamples()
	//podExamples()
	//configMapExamples()
	secretExamples()
}

func secretExamples() {
	var ( // {{{
		name          = "se"
		filepath      = "./examples/secret.yaml"
		labelSelector = "type=secret"
		forceDelete   = false
	)

	secret, err := k8sclient.NewSecret(ctx, namespace, *kubeconfig)
	if err != nil {
		logrus.Error(err)
	}
	_ = name
	_ = filepath
	_ = labelSelector
	_ = secret
	_ = forceDelete

	// 1. create secret
	if s, err := secret.Create(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("create secret %s success.", s.Name)
	}
	// 2. update secret
	if s, err := secret.Update(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("update secret %s success.", s.Name)
	}
	// 3. apply secret
	if s, err := secret.Apply(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("apply secret %s success.", s.Name)
	}
	secret.Delete(name, forceDelete)
	if s, err := secret.Apply(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("apply secret %s success.", s.Name)
	}
	// 4. delete secret
	if err := secret.Delete(name, forceDelete); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("delete secret %s success.", name)
	}
	// 5. get secret
	secret.Create(filepath)
	if s, err := secret.Get(name); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("get secret %s success.", s.Name)
	}
	// 6. list secret
	if cl, err := secret.List(labelSelector); err != nil {
		logrus.Error(err)
	} else {
		logrus.Info("list secret success.")
		for _, s := range cl.Items {
			logrus.Info(s.Name)
		}
	}
	// 7. watch secret
	secret.Watch(labelSelector,
		func() {
			logrus.Info("add secret.")
		},
		func() {
			logrus.Info("modified secret.")
		},
		func() {
			logrus.Info("delete secret")
		},
	)
} // }}}
func configMapExamples() {
	var ( // {{{
		name          = "cm"
		filepath      = "./examples/cm.yaml"
		labelSelector = "type=config"
		forceDelete   = false
	)

	configMap, err := k8sclient.NewConfigMap(ctx, namespace, *kubeconfig)
	if err != nil {
		logrus.Error(err)
	}
	_ = name
	_ = filepath
	_ = labelSelector
	_ = configMap
	_ = forceDelete

	// 1. create configmap
	if c, err := configMap.Create(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("create configMap %s success.", c.Name)
	}
	// 2. update configmap
	if c, err := configMap.Update(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("update configMap %s success.", c.Name)
	}
	// 3. apply configmap
	if c, err := configMap.Apply(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("apply configmap %s success.", c.Name)
	}
	configMap.Delete(name, forceDelete)
	if c, err := configMap.Apply(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("apply configmap %s success.", c.Name)
	}
	// 4. delete configmap
	if err := configMap.Delete(name, forceDelete); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("delete configmap %s success.", name)
	}
	// 5. get configmap
	configMap.Create(filepath)
	if c, err := configMap.Get(name); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("get configmap %s success.", c.Name)
	}
	// 6. list configmap
	if cl, err := configMap.List(labelSelector); err != nil {
		logrus.Error(err)
	} else {
		logrus.Info("list configMap success.")
		for _, c := range cl.Items {
			logrus.Info(c.Name)
		}
	}
	// 7. watch configmap
	configMap.Watch(labelSelector,
		func() {
			logrus.Info("add configmap.")
		},
		func() {
			logrus.Info("modified configmap.")
		},
		func() {
			logrus.Info("delete configmap")
		},
	)
} // }}}
func podExamples() {
	var ( // {{{
		name          = "pod-nginx"
		labelSelector = "app=pod-nginx"
		filepath      = "./examples/pod-nginx.yaml"
		forceDelete   = false
	)

	pod, err := k8sclient.NewPod(ctx, namespace, *kubeconfig)
	if err != nil {
		logrus.Error(err)
	}
	_ = name
	_ = labelSelector
	_ = pod
	_ = filepath
	_ = forceDelete

	// 1. create pod
	if p, err := pod.Create(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("create pod %s success.", p.Name)
	}

	//// 2. update pod
	//// pod 很多字段都是不可修改的, 更新 pod 往往会失败, 这里就不演示了
	//if p, err := pod.Update(filepath); err != nil {
	//    logrus.Error(err)
	//} else {
	//    logrus.Infof("update pod %s success.", p.Name)
	//}

	pod.Delete(name, forceDelete)
	// 3. apply pod
	if p, err := pod.Apply(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("apply pod %s success", p.Name)
	}

	// 4. delete pod
	if err := pod.Delete(name, forceDelete); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("delete pod %s success", name)
	}
	// 5. get pod
	pod.Apply(filepath)
	if p, err := pod.Get(name); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("get pod %s success.", p.Name)
	}
	// 6. list pod
	if pl, err := pod.List(labelSelector); err != nil {
		logrus.Error(err)
	} else {
		logrus.Info("list pod success.")
		for _, p := range pl.Items {
			logrus.Info(p.Name)
		}
	}
	// 7. watch pod
	pod.Watch(labelSelector,
		func() {
			logrus.Info("add pod.")
		},
		func() {
			logrus.Info("modified pod.")
		},
		func() {
			logrus.Info("deleted pod.")
		},
	)

} // }}}
func nodeExamples() {
	var ( // {{{
		name = "d11-k8s-worker3"
		//labelSelector = ""
		labelSelector = "node-role.kubernetes.io/master"
	)
	node, err := k8sclient.NewNode(ctx, *kubeconfig)
	if err != nil {
		logrus.Error(err)
	}
	_ = node
	_ = name
	_ = labelSelector

	// 1. get node
	if n, err := node.Get(name); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("get node %s success.", n.Name)
	}
	// 2. list node
	if nl, err := node.List(labelSelector); err != nil {
		logrus.Error(err)
	} else {
		logrus.Info("list node success.")
		for _, n := range nl.Items {
			logrus.Info(n.Name)
		}
	}
} // }}}

func deploymentWatcher() {
	var ( // {{{
		name          = "k8s-tools"
		filepath      = "./examples/k8s-tools.yaml"
		forceDelete   = false
		labelSelector = "app=k8s-tools"
	)

	deployment, err := k8sclient.NewDeployment(ctx, namespace, *kubeconfig)
	if err != nil {
		logrus.Fatal(err)
	}
	_ = ctx
	_ = name
	_ = filepath
	_ = forceDelete
	_ = deployment
	_ = labelSelector

	// 7. watch deployment
	logrus.Infof("start watch %q", labelSelector)
	deployment.Watch(labelSelector,
		func() {
			logrus.Info("add deployment.")
		},
		func() {
			logrus.Info("modified deployment.")
		},
		func() {
			logrus.Info("deleted deployment.")
		},
	)
} // }}}
func deploymentExamples() {
	var ( // {{{
		name          = "k8s-tools"
		filepath      = "./examples/k8s-tools.yaml"
		forceDelete   = false
		labelSelector = "app=k8s-tools"
	)

	deployment, err := k8sclient.NewDeployment(ctx, namespace, *kubeconfig)
	if err != nil {
		logrus.Fatal(err)
	}
	_ = name
	_ = filepath
	_ = deployment
	_ = labelSelector

	// 1. create deployment
	if deploy, err := deployment.Create(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("create deployment %s success.", deploy.Name)
	}
	// 2. Update deployment
	if deploy, err := deployment.Update(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("update deployment %s success.", deploy.Name)
	}
	// 3. apply deployment
	if deploy, err := deployment.Apply(filepath); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("apply deployment %s success.", deploy.Name)
	}
	deployment.Delete(name, forceDelete)
	deployment.Apply(filepath)
	// 4. Delete deployment
	if err := deployment.Delete(name, forceDelete); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("delete deployment %s success.", name)
	}
	deployment.Create(filepath)
	// 5. get deployment
	if deploy, err := deployment.Get(name); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("get deployment %s success.", deploy.Name)
	}
	// 6. list deployment
	if deployList, err := deployment.List(labelSelector); err != nil {
		logrus.Error(err)
	} else {
		logrus.Infof("list deployments by label: %s success.", labelSelector)
		for _, deploy := range deployList.Items {
			logrus.Info(deploy.Name)
		}
	}
	// 7. watch deployment
	deployment.Watch(labelSelector,
		func() {
			logrus.Info("add deployment.")
		},
		func() {
			logrus.Info("modified deployment.")
		},
		func() {
			logrus.Info("deleted deployment.")
		},
	)

} // }}}

//func dynamicPodsExamples() {
//    var ( // {{{
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
//        logrus.Fatal()
//    }

//    // 1. create pod
//    if pod, err := k8sdynamic.CreatePods(ctx, client, namespace, filepath); err != nil {
//        logrus.Error(err)
//    } else {
//        logrus.Info("create pod success.")
//        logrus.Info(pod.Name)
//    }

//    //// 2. Update pod
//    //// pod 几乎所有字段都是不可变的, 更新 pod 往往会失败, 所有不演示 update pod
//    //pod, err = k8sdynamic.UpdatePods(ctx, client, namespace, filepath)
//    //if err != nil {
//    //    logrus.Error(err)
//    //} else {
//    //    logrus.Info("update pod success.")
//    //    logrus.Info(pod.Name)
//    //}

//    // 3. delete pod
//    if err := k8sdynamic.DeletePod(ctx, client, namespace, name, forceDelete); err != nil {
//        logrus.Error(err)
//    } else {
//        logrus.Info("delete pod success.")
//    }
//    if _, err := k8sdynamic.CreatePods(ctx, client, namespace, filepath); err != nil {
//        logrus.Error()
//    }

//    // 4. Get pod
//    if pod, err := k8sdynamic.GetPod(ctx, client, namespace, name); err != nil {
//        logrus.Error(err)
//    } else {
//        logrus.Info("get pod success.")
//        logrus.Info(pod.Name)
//    }

//    // 5. list pods
//    if podList, err := k8sdynamic.ListPods(ctx, client, namespace, labelSelector); err != nil {
//        logrus.Error(err)
//    } else {
//        logrus.Info("list pods success.")
//        for _, pod := range podList.Items {
//            logrus.Info(pod.Name)
//        }
//    }

//    // 6. watch pods
//    k8sdynamic.WatchPods(ctx, client, namespace, labelSelector)
//} // }}}
//func dynamicDeploymentsExamples() {
//    var ( // {{{
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
//        logrus.Fatal()
//    }

//    // 1. create deployment
//    if deploy, err := k8sdynamic.CreateDeployment(ctx, client, namespace, filepath); err != nil {
//        logrus.Error(err)
//    } else {
//        logrus.Info("create deployment success.")
//        logrus.Info(deploy.Name)
//    }

//    // 2. update deployment
//    if deploy, err := k8sdynamic.UpdateDeployment(ctx, client, namespace, filepath); err != nil {
//        logrus.Error(err)
//    } else {
//        logrus.Info("update deployment success.")
//        logrus.Info(deploy.Name)
//    }

//    // 3. delete deployment
//    if err = k8sdynamic.DeleteDeployment(ctx, client, namespace, name, forceDelete); err != nil {
//        logrus.Error(err)
//    } else {
//        logrus.Info("delete deployment k8s-tools success")
//    }
//    if _, err = k8sdynamic.CreateDeployment(ctx, client, namespace, filepath); err != nil {
//        logrus.Error(err)
//    }

//    // 4. get deployment
//    if deploy, err := k8sdynamic.GetDeployment(ctx, client, namespace, name); err != nil {
//        logrus.Error(err)
//    } else {
//        logrus.Info("get deployment success.")
//        logrus.Info(deploy.GetName())
//        logrus.Info(*deploy.Spec.Replicas)
//    }

//    // 5. list deployment
//    if deployList, err := k8sdynamic.ListDepoyments(ctx, client, namespace, labelSelector); err != nil {
//        logrus.Error(err)
//    } else {
//        logrus.Info("list deployments success.")
//        for _, deploy := range deployList.Items {
//            logrus.Info(deploy.Name)
//        }
//    }

//    // 6. watch deployment
//    k8sdynamic.WatchDeployments(ctx, client, namespace, labelSelector)
//} // }}}
