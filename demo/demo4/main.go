package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config := "config"
	namespace := "default"
	var (
		clientset     *kubernetes.Clientset
		deployYaml    []byte
		deployJson    []byte
		deployment    = apps_v1.Deployment{}
		k8sDeployment *apps_v1.Deployment
		pod           core_v1.Pod
		podList       *core_v1.PodList
		err           error
	)

	// 1. 初始化客户端
	if clientset, err = InitClientset(config); err != nil {
		goto FAIL
	}
	// 2. 读取 yaml 文件
	if deployYaml, err = ioutil.ReadFile("./nginx.yaml"); err != nil {
		goto FAIL
	}
	// 3. yaml 转 json
	if deployJson, err = yaml.ToJSON(deployYaml); err != nil {
		goto FAIL
	}
	// 4. Json 转 struct
	if err = json.Unmarshal(deployJson, &deployment); err != nil {
		goto FAIL
	}
	// 5. 更新 deployment
	// 给 Pod 增加 label
	deployment.Spec.Template.Labels["deploy_time"] = strconv.Itoa(int(time.Now().Unix()))
	if _, err := clientset.AppsV1().Deployments(namespace).Update(context.Background(),
		&deployment, meta_v1.UpdateOptions{}); err != nil {
		goto FAIL
	}

	// 6. 等待更新完成
	for {
		// 获取 k8s 中 deployment 的状态
		if k8sDeployment, err = clientset.AppsV1().Deployments(namespace).Get(context.Background(),
			deployment.Name, meta_v1.GetOptions{}); err != nil {
			goto RETRY
		}
		// 进行状态判定
		if k8sDeployment.Status.UpdatedReplicas == *(k8sDeployment.Spec.Replicas) &&
			k8sDeployment.Status.Replicas == *(k8sDeployment.Spec.Replicas) &&
			k8sDeployment.Status.AvailableReplicas == *(k8sDeployment.Spec.Replicas) &&
			k8sDeployment.Status.ObservedGeneration == k8sDeployment.Generation {
			break // 滚动升级完成
		}
		// 打印工作中的 pod 比例
		fmt.Printf("部署中: (%d/%d)\n", k8sDeployment.Status.AvailableReplicas, *(k8sDeployment.Spec.Replicas))
	RETRY:
		time.Sleep(time.Second)
	}

	// 7. 打印每个 pod 的状态(可能会打印出 terminating 中的 pod, 但最终只会展示新 pod 列表)
	if podList, err = clientset.CoreV1().Pods(namespace).List(context.Background(),
		meta_v1.ListOptions{LabelSelector: "app=nginx"}); err == nil {
		for _, pod = range podList.Items {
			podName := pod.Name
			podStatus := string(pod.Status.Phase)

			// PodRunning means the pod has been bound to a node and all of the containers have been started.
			// At least one container is still running or is in the process of being restarted.
			if podStatus == string(core_v1.PodRunning) {
				// 汇总错误原因不为空
				if pod.Status.Reason != "" {
					podStatus = pod.Status.Reason
					goto KO
				}
				// condition 有错误信息
				for _, cond := range pod.Status.Conditions {
					if cond.Type == core_v1.PodReady { // Pod 就绪状态
						if cond.Status != core_v1.ConditionTrue { // 失败
							podStatus = cond.Reason
						}
						goto KO
					}
				}
				// 没有ready condition, 状态未知
				podStatus = "Unknown"
			}

		KO:
			fmt.Printf("[name:%s status:%s]\n", podName, podStatus)
		}
	}

	return

FAIL:
	log.Println(err)
	return
}

func InitClientset(config string) (clietset *kubernetes.Clientset, err error) {
	var (
		kubeConfig []byte
		restConfig *rest.Config
	)

	// 1. get kubeConfig
	if kubeConfig, err = ioutil.ReadFile(config); err != nil {
		goto FAIL
	}
	// 2. get restConfig
	if restConfig, err = clientcmd.RESTConfigFromKubeConfig(kubeConfig); err != nil {
		goto FAIL
	}
	// 3. get clientset
	if clietset, err = kubernetes.NewForConfig(restConfig); err != nil {
		goto FAIL
	}

	return

FAIL:
	log.Println(err)
	return
}
