package dynamic

// refernces:
//	https://github.com/kubernetes/client-go/blob/master/examples/dynamic-create-update-delete-deployment/main.go

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
)

// CreatePod 从 yaml 文件中读取, 通过 dynamic 客户端创建 pod. 转换过程如下:
// file -> yaml -> json -> corev1.Pod{} -> map[string]interface{} -> unstructured.Unstructured{}
func CreatePods(ctx context.Context, client dynamic.Interface, namespace, filepath string) (pod *corev1.Pod, err error) {
	var (
		gvr         = schema.GroupVersionResource{Version: "v1", Resource: "pods"}
		podYaml     []byte
		podJson     []byte
		object      map[string]interface{}
		unstructObj = &unstructured.Unstructured{}
	)
	pod = &corev1.Pod{}                                       // 必须先初始化一下
	if podYaml, err = ioutil.ReadFile(filepath); err != nil { // file -> yaml
		return
	}
	if podJson, err = yaml.ToJSON(podYaml); err != nil { // yaml -> json
		return
	}
	if err = json.Unmarshal(podJson, pod); err != nil { // json -> corev1.Pod{}
		return
	}
	if object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(pod); err != nil { // appsv1.Deployment{} -> map[string]interface{}
		return
	}
	unstructObj = &unstructured.Unstructured{Object: object} // map[string]interface{} -> unstructured.Unstructured{}

	if _, err = client.Resource(gvr).Namespace(namespace).Create(ctx, unstructObj, metav1.CreateOptions{}); err != nil {
		return
	}
	return
}

//// UpdatePod 从 yaml 文件中读取, 通过 dynamic 客户端创建 pod. 转换过程如下:
//// file -> yaml -> json -> corev1.Pod{} -> map[string]interface{} -> unstructured.Unstructured{}
func UpdatePods(ctx context.Context, client dynamic.Interface, namespace, filepath string) (pod *corev1.Pod, err error) {
	var (
		gvr         = schema.GroupVersionResource{Version: "v1", Resource: "pods"}
		podYaml     []byte
		podJson     []byte
		object      map[string]interface{}
		unstructObj = &unstructured.Unstructured{}
	)
	pod = &corev1.Pod{}                                       // 必须先初始化
	if podYaml, err = ioutil.ReadFile(filepath); err != nil { // file -> yaml
		return
	}
	if podJson, err = yaml.ToJSON(podYaml); err != nil { // yaml -> json
		return
	}
	if err = json.Unmarshal(podJson, pod); err != nil { // json -> corev1.Pod{}
		return
	}
	if object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(pod); err != nil { // corev1.Pod{} -> map[string]interface{}
		return
	}
	unstructObj = &unstructured.Unstructured{Object: object} // map[string]interface{} -> unstructured.Unstructured{}
	if _, err = client.Resource(gvr).Namespace(namespace).Update(ctx, unstructObj, metav1.UpdateOptions{}); err != nil {
		return
	}
	return
}

// DeletePod 通过 dynamic 客户端删除 pod.
func DeletePod(ctx context.Context, client dynamic.Interface, namespace, name string, forceDelete bool) (err error) {
	var (
		gvr                = schema.GroupVersionResource{Version: "v1", Resource: "pods"}
		gracePeriodSeconds int64
	)
	if forceDelete {
		gracePeriodSeconds = 0
	}
	if err = client.Resource(gvr).Namespace(namespace).Delete(ctx,
		name, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds}); err != nil {
		return
	}
	return
}

// Getpod 通过 dynamic 客户端获取 pod. 转换过程如下:
// unstructured.Unstructured{} -> corev1.Pod{}
func GetPod(ctx context.Context, client dynamic.Interface, namespace, name string) (pod *corev1.Pod, err error) {
	var (
		gvr         = schema.GroupVersionResource{Version: "v1", Resource: "pods"}
		unstructObj = &unstructured.Unstructured{}
	)
	pod = &corev1.Pod{} // deploy 一定要先初始化
	if unstructObj, err = client.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{}); err != nil {
		return
	}
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructObj.UnstructuredContent(), pod); err != nil {
		return
	}
	return
}

// ListPods 通过 dynamic 客户端获取 pod 列表.
func ListPods(ctx context.Context, client dynamic.Interface, namespace, labelSelector string) (podList *corev1.PodList, err error) {
	var (
		gvr              = schema.GroupVersionResource{Version: "v1", Resource: "pods"}
		unstructuredList = &unstructured.UnstructuredList{}
		timeoutSeconds   = int64(0)
		limit            = int64(100)
	)
	podList = &corev1.PodList{} // 必须要先初始化一下
	if unstructuredList, err = client.Resource(gvr).Namespace(namespace).List(ctx,
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeoutSeconds, Limit: limit}); err != nil {
		return
	}
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredList.UnstructuredContent(), podList); err != nil {
		return
	}
	return
}

// WatchPods 通过 dynamic 客户端监控 pod 事件
func WatchPods(ctx context.Context, client dynamic.Interface, namespace, labelSelector string) {
	var (
		gvr   = schema.GroupVersionResource{Version: "v1", Resource: "pods"}
		mutex *sync.Mutex
	)
	for {
		watch, err := client.Resource(gvr).Namespace(namespace).Watch(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			logrus.Error(err)
			return
		}
		watchHandler(watch.ResultChan(), mutex)
	}
}

//// CreatePodDemo 直接通过 dynamic 客户端创建 pod ,不需要文件.
//func CreatePodDemo(ctx context.Context, client dynamic.Interface, namespace, filepath string) (pod *corev1.Pod, err error) {
//    deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
//    deployment := &unstructured.Unstructured{
//        Object: map[string]interface{}{
//            "apiVerion": "apps/v1",
//            "kind:":     "Deployment",
//            "metadata": map[string]interface{}{
//                "name": "demo-deployment",
//            },
//            "spec": map[string]interface{}{
//                "replicas": 2,
//                "selector": map[string]interface{}{
//                    "matchLabels": map[string]interface{}{
//                        "app": "demo",
//                    },
//                },
//                "template": map[string]interface{}{
//                    "metadata": map[string]interface{}{
//                        "labels": map[string]interface{}{
//                            "app": "demo",
//                        },
//                    },
//                    "spec": map[string]interface{}{
//                        "containers": []map[string]interface{}{
//                            {
//                                "name":  "web",
//                                "image": "nginx:1.12",
//                                "ports": []map[string]interface{}{
//                                    {
//                                        "name":          "http",
//                                        "protocol":      "TCP",
//                                        "containerPort": 80,
//                                    },
//                                },
//                            },
//                        },
//                    },
//                },
//            },
//        },
//    }

//    // Create Deployment
//    logrus.Info("Creating deployment...")
//    result, err := client.Resource(deploymentRes).Namespace(namespace).Create(ctx, deployment, metav1.CreateOptions{})
//    if err != nil {
//        return nil, err
//    }
//    logrus.Infof("Created deployment %q.\n", result.GetName())
//    err = runtime.DefaultUnstructuredConverter.FromUnstructured(deployment.UnstructuredContent(), deploy)
//    if err != nil {
//        return nil, err
//    }

//    return deploy, err
//}
