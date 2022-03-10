package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	yaml2 "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

/*
	解析 yaml 为 json, 反序列化到 deployment 对象, 修改 deployment.spec.replicas, 提交到k8s生效
*/
func main() {
	config := "config"
	var (
		clientset  *kubernetes.Clientset
		podsList   *core_v1.PodList
		deployYaml []byte
		deployJson []byte
		deployment = apps_v1.Deployment{}
		replicas   int32
		err        error
	)

	// 初始化 k8s 客户端
	if clientset, err = initClient(config); err != nil {
		log.Println(err)
		return
	}
	// 获取 pods
	if podsList, err = clientset.CoreV1().Pods("default").List(context.Background(), meta_v1.ListOptions{}); err != nil {
		log.Println(err)
		return
	}
	_ = *podsList

	// 读取 yaml
	if deployYaml, err = ioutil.ReadFile("./nginx.yaml"); err != nil {
		log.Println(err)
		return
	}

	// YAML 转 JSON
	if deployJson, err = yaml2.ToJSON(deployYaml); err != nil {
		log.Println(err)
		return
	}

	// JSON 转 struct
	if err = json.Unmarshal(deployJson, &deployment); err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(deployYaml))
	fmt.Println(string(deployJson))

	// 修改 replicas 数量为 1
	replicas = 1

	// 查询k8s是否有该deployment
	deployment.Spec.Replicas = &replicas
	if _, err = clientset.AppsV1().Deployments("default").Get(context.Background(),
		deployment.Name, meta_v1.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			log.Println(err)
			return
		}
		// 不存在则更新
		if _, err = clientset.AppsV1().Deployments("default").Create(context.Background(),
			&deployment, meta_v1.CreateOptions{}); err != nil {
			log.Println(err)
			return
		}
	} else { // 已存在则创建
		if _, err = clientset.AppsV1().Deployments("default").Update(context.Background(),
			&deployment, meta_v1.UpdateOptions{}); err != nil {
			log.Println(err)
			return
		}
	}
	fmt.Println("deployment 创建成功!")
	return
}

func initClient(config string) (clientset *kubernetes.Clientset, err error) {
	var kubeconfig []byte
	var restConfig *rest.Config

	// kubeconfig --> restConfig
	// 先获取 kubeconfig 配置文件
	if kubeconfig, err = ioutil.ReadFile(config); err != nil {
		log.Println(err)
		return
	}
	// 从 kubeconfig 文件获取 restConfig 文件
	if restConfig, err = clientcmd.RESTConfigFromKubeConfig(kubeconfig); err != nil {
		log.Println(err)
		return
	}

	// restConfig --> clientset
	// 从 restConfig 获取 clientset 文件
	if clientset, err = kubernetes.NewForConfig(restConfig); err != nil {
		log.Println(err)
		return
	}

	return
}
