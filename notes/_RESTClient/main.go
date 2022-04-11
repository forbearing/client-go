package main

import (
	"context"
	"fmt"
	"log"
	"os"

	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

/*
	- RESTClient 是最基础的客户端，其他的ClientSet、DynamicClient、DiscoveryClient
	  都是基于 RESTClient 实现的。RESTClient 对 HTTP Request 进行了封装，实现了 RESTful
	  风格的 API。
*/

func main() {
	var (
		restClient *rest.RESTClient
		podList    = &core_v1.PodList{}
		err        error
	)
	namespace := "kube-system"

	// 1. 获得 restClient
	if restClient, err = NewRESTClient(); err != nil {
		goto FAIL
	}

	// 2. 操作 restClient，获得 pod 列表
	// 构建 HTTP 请求参数
	// 请求方法可以是 Get、Post、Put、Delete、Patch 等
	// Namespace 函数设置请求的命名空间
	// Resource 函数设置请求的资源名称
	// VersionParams 函数将一些查询选项（如limit、TimeoutSeconds等）添加到请求参数中
	// Do 函数执行请求，并将 kube-apiserver 返回的 result(这里是 podList) 对象解析到 corev1.PodList 对象中
	if err = restClient.Get().Namespace(namespace).Resource("pods").
		VersionedParams(&meta_v1.ListOptions{Limit: 500}, scheme.ParameterCodec).
		Do(context.Background()).Into(podList); err != nil {
		goto FAIL
	}

	// 3. 打印 pod 列表
	for _, pod := range podList.Items {
		fmt.Printf("NAMESPACE:%v \t NAME:%v \t STATU:%v\n",
			pod.Namespace, pod.Name, pod.Status.Phase)
	}

	return

FAIL:
	log.Println(err)
	return

}

func NewRESTClient() (restClient *rest.RESTClient, err error) {
	var (
		kubeConfigPath string
		restConfig     *rest.Config
	)

	// 1. 获取 kubeconfig 文件绝对路径
	kubeConfigPath = os.Getenv("HOME") + "/.kube/config"

	// 2. 获取 restConfig
	//	通过 clientcmd.BuildConfigFromFlags 从 kubeConfigPath 获得 restConfig
	if restConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath); err != nil {
		goto FAIL
	}

	// 3. 配置 restConfig
	restConfig.APIPath = "api"                            // 设置restConfig.APIPath 请求的 HTTP 路径
	restConfig.GroupVersion = &core_v1.SchemeGroupVersion // 设置 restConfig.GroupVersion 请求的资源组/资源版本
	restConfig.NegotiatedSerializer = scheme.Codecs       // 设置 restConfig.NegotiatedSerializer 数据的解码器

	// 4. 获取 RESTClient
	//	通过 rest.RESTClientFor 从 restConfig 获得 restClient
	if restClient, err = rest.RESTClientFor(restConfig); err != nil {
		goto FAIL
	}
	return

FAIL:
	log.Println(err)
	return

}
