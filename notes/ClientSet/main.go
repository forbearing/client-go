package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

/*
- RESTClient 是最基础的客户端，使用时需要指定 Resource 和 Version 等信息，编写代码时需
  要提前知道 Resource 所在的 Group 和对应的 Version 信息。ClientSet 相比而言使用更加便捷，
  一般情况，对 Kubernetes 进行二次开发时通常使用 ClientSet。 ClientSet 在 RESTClient 的
  基础上封装了对 Resource 和 Version 的管理方法，每个 Resource 可以理解为一个客户端，而
  ClientSet 则是多个客户端的集合，每个 Resource 和 Version 都以函数的方式暴露给开发者。
- ClientSet 仅能访问 Kubernetes 自身的内置资源，不能直接访问 CRD 自定义资源；如果需要
  使用 ClientSet 访问 CRD，则需要通过 client-gen 代码生成器重新生成 ClientSet；
- DynamicClient 可以访问CRD资源
*/

func main() {
	podHandler()
}

// 这个函数是用来 Apply, Create, Delete, Get, List, Watch Pod 的
// 后续加入更多的功能
func podHandler() {
	var (
		clientset *kubernetes.Clientset
		err       error
		pod       *core_v1.Pod
		podList   *core_v1.PodList
	)
	namespace := "default"
	podName := "pod-nginx"

	// 1. 获取 clientset
	if clientset, err = NewClientSet(); err != nil {
		log.Println("NewClientSet error")
		goto FAIL
	}

	// 2. get pod
	if pod, err = clientset.CoreV1().Pods(namespace).Get(context.Background(),
		podName, meta_v1.GetOptions{}); err != nil {
		log.Println("clientset error")
		goto FAIL
	}

	// 3. list pod
	if podList, err = clientset.CoreV1().Pods(core_v1.NamespaceDefault).
		List(context.Background(), meta_v1.ListOptions{Limit: 500}); err != nil {
		goto FAIL
	}

	// 4. output pods info
	fmt.Println(pod.Status.String())
	for _, value := range podList.Items {
		fmt.Printf("NAMESPACE:%v \t NAME:%v \t STATU:%v\n",
			value.Namespace, value.Name, value.Status.Phase)
	}

FAIL:
	log.Println(err)
	return
}

func NewClientSet() (clientset *kubernetes.Clientset, err error) {
	var (
		kubeConfig     []byte
		kubeConfigPath string
		restConfig     *rest.Config
		restConfig2    *rest.Config
	)

	// 1. 获取 kubeConfig 文件的绝对路径
	kubeConfigPath = os.Getenv("HOME") + "/.kube/config"

	// 2. 通过 kubeConfig 获取 restclient 对象
	//	restclient 提供 RESTClient 客户端，对 Kuberntes API Server 执行 RESTful 操作
	//	ClientSet、DynamicClient、DiscoveryClient 客户端都是基于 RESTClient 实现的

	// 方法一: 直接从 kubeconfig 文件中读取得到 restConfig
	if restConfig2, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath); err != nil {
		goto FAIL
	}
	_ = restConfig2
	// 方法二: 先将 kubeconfig 读取出来，再得到 restConfig
	if kubeConfig, err = ioutil.ReadFile(kubeConfigPath); err != nil {
		goto FAIL
	}
	if restConfig, err = clientcmd.RESTConfigFromKubeConfig(kubeConfig); err != nil {
		goto FAIL
	}

	// 3. 通过 restclient 对象获取 clientset 对象
	if clientset, err = kubernetes.NewForConfig(restConfig); err != nil {
		goto FAIL
	}

	return
FAIL:
	log.Println(err)
	return
}
