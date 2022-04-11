package dynamic

// refernces:
//	https://github.com/kubernetes/client-go/blob/master/examples/dynamic-create-update-delete-deployment/main.go

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
)

// CreateDeployment 从 yaml 文件中读取, 通过 dynamic 客户端创建 deployment. 转换过程如下:
// file -> yaml -> json -> appsv1.Deployment{} -> map[string]interface{} -> unstructured.Unstructured{}
func CreateDeployment(ctx context.Context, client dynamic.Interface, namespace, filepath string) (deploy *appsv1.Deployment, err error) {
	var (
		gvr         = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		deployYaml  []byte
		deployJson  []byte
		unstructMap map[string]interface{}
		unstructObj = &unstructured.Unstructured{}
	)
	deploy = &appsv1.Deployment{}                                // 必须先初始化
	if deployYaml, err = ioutil.ReadFile(filepath); err != nil { // file -> yaml
		return
	}
	if deployJson, err = yaml.ToJSON(deployYaml); err != nil { // yaml -> json
		return
	}
	if err = json.Unmarshal(deployJson, deploy); err != nil { // json -> appsv1.Deployment{}
		return
	}
	if unstructMap, err = runtime.DefaultUnstructuredConverter.ToUnstructured(deploy); err != nil { // appsv1.Deployment{} -> map[string]interface{}
		return
	}
	unstructObj = &unstructured.Unstructured{Object: unstructMap} // map[string]interface{} -> unstructured.Unstructured{}
	if _, err = client.Resource(gvr).Namespace(namespace).Create(ctx, unstructObj, metav1.CreateOptions{}); err != nil {
		return
	}
	return
}

// UpdateDeployment 从 yaml 文件中读取, 通过 dynamic 客户端创建 deployment. 转换过程如下:
// file -> yaml -> json -> appsv1.Deployment{} -> map[string]interface{} -> unstructured.Unstructured{}
func UpdateDeployment(ctx context.Context, client dynamic.Interface, namespace, filepath string) (deploy *appsv1.Deployment, err error) {
	var (
		gvr         = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		deployYaml  []byte
		deployJson  []byte
		unstructMap map[string]interface{}
		unstructObj = &unstructured.Unstructured{}
	)
	deploy = &appsv1.Deployment{}                                // 必须先初始化
	if deployYaml, err = ioutil.ReadFile(filepath); err != nil { // file -> yaml
		return
	}
	if deployJson, err = yaml.ToJSON(deployYaml); err != nil { // yaml -> json
		return
	}
	if err = json.Unmarshal(deployJson, deploy); err != nil { // json -> appsv1.Deployment{}
		return
	}
	if unstructMap, err = runtime.DefaultUnstructuredConverter.ToUnstructured(deploy); err != nil { // appsv1.Deployment{} -> map[string]interface{}
		return
	}
	unstructObj = &unstructured.Unstructured{Object: unstructMap} // map[string]interface{} -> unstructured.Unstructured{}
	if _, err = client.Resource(gvr).Namespace(namespace).Update(ctx, unstructObj, metav1.UpdateOptions{}); err != nil {
		return
	}
	return
}

// DeleteDeployment 通过 dynamic 客户端删除 deployment.
func DeleteDeployment(ctx context.Context, client dynamic.Interface, namespace, name string, forceDelete bool) (err error) {
	var (
		gracePeriodSeconds int64
		gvr                = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
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

// GetDeployment 通过 dynamic 客户端获取 deployment. 转换过程如下:
// unstructured.Unstructured{} -> appsv1.Deployment{}
func GetDeployment(ctx context.Context, client dynamic.Interface, namespace, name string) (deploy *appsv1.Deployment, err error) {
	var (
		gvr         = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		unstructObj = &unstructured.Unstructured{}
		//unstructMap      map[string]interface{}
	)
	deploy = &appsv1.Deployment{} // deploy 一定要先初始化
	if unstructObj, err = client.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{}); err != nil {
		return
	}
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructObj.UnstructuredContent(), deploy); err != nil {
		return
	}
	return
}

// ListDepoyments 通过 dynamic 客户端获取 deployment 列表.
func ListDepoyments(ctx context.Context, client dynamic.Interface, namespace, labelSelector string) (deployList *appsv1.DeploymentList, err error) {
	var (
		gvr              = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		unstructuredList = &unstructured.UnstructuredList{}
		timeoutSeconds   = int64(0)
		limit            = int64(100)
	)
	deployList = &appsv1.DeploymentList{} // 必须要先初始化一下
	if unstructuredList, err = client.Resource(gvr).Namespace(namespace).List(ctx,
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeoutSeconds, Limit: limit}); err != nil {
		return
	}
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredList.UnstructuredContent(), deployList); err != nil {
		return
	}
	return
}

// WatchDeployments 通过 dynamic 客户端监控 deployment 事件
func WatchDeployments(ctx context.Context, client dynamic.Interface, namespace, labelSelector string) {
	var (
		gvr   = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
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

// CreateDeploymentDemo 直接通过 dynamic 客户端创建 deployment ,不需要文件.
func CreateDeploymentDemo(ctx context.Context, client dynamic.Interface, namespace, filepath string) (deploy *appsv1.Deployment, err error) {
	deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVerion": "apps/v1",
			"kind:":     "Deployment",
			"metadata": map[string]interface{}{
				"name": "demo-deployment",
			},
			"spec": map[string]interface{}{
				"replicas": 2,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "demo",
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "demo",
						},
					},
					"spec": map[string]interface{}{
						"containers": []map[string]interface{}{
							{
								"name":  "web",
								"image": "nginx:1.12",
								"ports": []map[string]interface{}{
									{
										"name":          "http",
										"protocol":      "TCP",
										"containerPort": 80,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	logrus.Info("Creating deployment...")
	result, err := client.Resource(deploymentRes).Namespace(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	logrus.Infof("Created deployment %q.\n", result.GetName())
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(deployment.UnstructuredContent(), deploy)
	if err != nil {
		return nil, err
	}

	return deploy, err
}
