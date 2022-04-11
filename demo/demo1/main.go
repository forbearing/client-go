package main

import (
	"context"
	"fmt"
	"io/ioutil"

	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

/*
我的理解
	kubeconfig --> restConfig --> clientset
*/

func main() {
	var (
		clientset *kubernetes.Clientset
		podsList  *core_v1.PodList
		err       error
	)
	// 初始化 k8s 客户端
	if clientset, err = InitClient(); err != nil {
		goto FAIL
	}

	// 获取 default 命名空间下的所有 Pod
	if podsList, err = clientset.CoreV1().Pods("default").List(context.Background(), meta_v1.ListOptions{}); err != nil {
		goto FAIL
	}

	fmt.Println(*podsList)
	return

FAIL:
	fmt.Println(err)
	return
}

func InitClient() (clientset *kubernetes.Clientset, err error) {
	var restConf *rest.Config
	if restConf, err = GetRestConf(); err != nil {
		return
	}

	// 生成 clientset 配置
	if clientset, err = kubernetes.NewForConfig(restConf); err != nil {
		goto END
	}
END:
	return

}

func GetRestConf() (restConf *rest.Config, err error) {
	var kubeconfig []byte

	// 读取 kubeconfig 文件
	if kubeconfig, err = ioutil.ReadFile("config"); err != nil {
		goto END
	}

	// 生成 rest client 配置
	if restConf, err = clientcmd.RESTConfigFromKubeConfig(kubeconfig); err != nil {
		goto END
	}

END:
	return
}
