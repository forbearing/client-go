package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

/*
	功能
		获取 POD 内 container 的日志

	相关函数和方法
		request = clientset.CoreV1().Pods(namespace).GetLogs()
		result = request.Do(ctx)
		logs, err = result.Raw()
		result.Error()

*/

func main() {
	config := "config" // kubeconfig 文件，用来连接 k8s
	namespace := "kube-system"
	podName := "coredns-6688c55d47-hsrbw"
	containerName := "coredns"
	var (
		clientset *kubernetes.Clientset
		tailLines int64 = 10 // 获取最后10条日志
		request   *rest.Request
		result    rest.Result
		logs      []byte
		err       error
	)

	// 1. 获取 clientset
	if clientset, err = initClientset(config); err != nil {
		goto FAIL
	}
	// 2. 生成获取 POD 日志请求
	request = clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
	})

	// 3. 发送请求
	// req.Stream() 也可以实现 Do 的效果
	if result = request.Do(context.Background()); result.Error() != nil {
		err = result.Error()
		goto FAIL
	}

	// 4. 获取结果
	if logs, err = result.Raw(); err != nil {
		goto FAIL
	}

	fmt.Printf("pod(%s) 的容器(%s) 的最后 %v 行日志: \n%v\n",
		podName, containerName, tailLines, string(logs))
	return

FAIL:
	log.Println(err)
	return
}

func initClientset(config string) (clientset *kubernetes.Clientset, err error) {
	var (
		kubeConfig []byte
		restConfig *rest.Config
	)

	if kubeConfig, err = ioutil.ReadFile(config); err != nil {
		goto FAIL
	}
	if restConfig, err = clientcmd.RESTConfigFromKubeConfig(kubeConfig); err != nil {
		goto FAIL
	}
	if clientset, err = kubernetes.NewForConfig(restConfig); err != nil {
		goto FAIL
	}

	return

FAIL:
	log.Println(err)
	return
}
