package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	// yaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

/*
	更新 deployment.Spec.Template.Spec.Containers, 升级镜像版本, 提交到k8s生效
*/

func main() {
	config := "config"
	namespace := "default"
	var (
		clientset      *kubernetes.Clientset
		podsList       *core_v1.PodList
		deployYaml     []byte
		deployJson     []byte
		deployment     = apps_v1.Deployment{}
		containers     []core_v1.Container
		nginxContainer core_v1.Container
		err            error
	)
	// 1. 初始化 k8s 客户端
	if clientset, err = InitClientset(config); err != nil {
		goto FAIL
	}
	// 获取 pods
	if podsList, err = clientset.CoreV1().Pods(namespace).List(context.Background(), meta_v1.ListOptions{}); err != nil {
		goto FAIL
	}
	_ = podsList
	// 2. 读取 yaml
	if deployYaml, err = ioutil.ReadFile("nginx.yaml"); err != nil {
		goto FAIL
	}
	// 3. YAML 转 Json
	if deployJson, err = yaml.ToJSON(deployYaml); err != nil {
		goto FAIL
	}
	// 4. JSON 转 struct
	if err = json.Unmarshal(deployJson, &deployment); err != nil {
		goto FAIL
	}

	// 5. 配置 container/nginx
	// 定义的 container
	nginxContainer.Name = "nginx"
	nginxContainer.Image = "nginx:1.20"
	containers = append(containers, nginxContainer)

	// 6. 配置 deployment/nginx
	deployment.Spec.Template.Spec.Containers = containers

	// 7 .更新 deployment
	if _, err = clientset.AppsV1().Deployments(namespace).Update(context.Background(), &deployment, meta_v1.UpdateOptions{}); err != nil {
		goto FAIL
	}

	fmt.Println("apply deployment/nginx 成功!")
	return

FAIL:
	fmt.Println(err)
	return
}

func InitClientset(config string) (clientset *kubernetes.Clientset, err error) {
	var (
		kubeConfig []byte
		restConfig *rest.Config
	)
	// 1. 获取 kubeConfig 配置文件
	if kubeConfig, err = ioutil.ReadFile(config); err != nil {
		goto FAIL
	}
	// 2. 获取 restConfig 文件
	if restConfig, err = clientcmd.RESTConfigFromKubeConfig(kubeConfig); err != nil {
		goto FAIL
	}
	// 3. 获取 clientset
	if clientset, err = kubernetes.NewForConfig(restConfig); err != nil {
		goto FAIL
	}

	return

FAIL:
	fmt.Println(err)
	return
}
