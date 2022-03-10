package clientset

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Deployment struct {
	namespace string
	limit     int64
	timeout   int64
	ctx       context.Context
	client    *kubernetes.Clientset

	sync.Mutex
}

// New 创建一个 deployment 对象
func NewDeployment(ctx context.Context, namespace, kubeconfig string) (deployment *Deployment, err error) {
	var (
		config *rest.Config
		client *kubernetes.Clientset
	)
	deployment = &Deployment{}

	// create rest config
	if len(kubeconfig) != 0 {
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return
		}
	} else {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return
		}
	}

	// create the clientset
	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	deployment.namespace = namespace
	deployment.limit = 100
	deployment.timeout = 10
	deployment.ctx = ctx
	deployment.client = client

	return
}
func (d *Deployment) SetLimit(limit int64) {
	d.Lock()
	defer d.Unlock()
	d.limit = limit
}
func (d *Deployment) SetTimeout(timeout int64) {
	d.Lock()
	defer d.Unlock()
	d.timeout = timeout
}

// CreateDeployment 创建一个不存在的 deployment
// 1. 从 yaml 文件中读取并创建 deployment, 如果 yaml 文件中存在非 deployment 的 k8s 资源,
//    例如 service, pod 等. 就会创建失败. 即这个 yaml 文件有且仅有 deployment 的配置.
// 2. 如果 deployment 已存在, 再次调用 CreateDeployment 会失败, 报错如下:
//    “deployments.apps "DEPLOYMENT_NAME" already exists”
// 3. 考虑从 struct 来创建 deployment
func (d *Deployment) Create(filepath string) (deploy *appsv1.Deployment, err error) {
	var (
		deployYaml []byte
		deployJson []byte
	)
	deploy = &appsv1.Deployment{}
	if deployYaml, err = ioutil.ReadFile(filepath); err != nil { // read yaml file
		return
	}
	if deployJson, err = yaml.ToJSON(deployYaml); err != nil { // yaml to json
		return
	}
	if err = json.Unmarshal(deployJson, deploy); err != nil { // json to appv1.Deloyment
		return
	}
	deploy, err = d.client.AppsV1().Deployments(d.namespace).Create(d.ctx, deploy, metav1.CreateOptions{})
	return
}

// UpdateDeployment 更新一个已经存在的 deployment
// 1. 从 yaml 文件中读取并更新 deployment, 如果 yaml 文件中存在非 deployment 的 k8s 资源,
//    例如 service, pod 等, 就会创建失败. 即这个 yaml 文件有且仅有 deployment 的配置.
// 2. 如果 deployment 不存在, UpdateDeployment 就会失败.
// 3. 即使 deployment 已经存在, 重复调用 UpdateDeployment 并不会报错.
func (d *Deployment) Update(filepath string) (deploy *appsv1.Deployment, err error) {
	var (
		deployYaml []byte
		deployJson []byte
	)
	deploy = &appsv1.Deployment{}
	if deployYaml, err = ioutil.ReadFile(filepath); err != nil { // read yaml file
		return
	}
	if deployJson, err = yaml.ToJSON(deployYaml); err != nil { // yaml to json
		return
	}
	if err = json.Unmarshal(deployJson, deploy); err != nil { // json to appv1.Deployment
		return
	}
	deploy, err = d.client.AppsV1().Deployments(d.namespace).Update(d.ctx, deploy, metav1.UpdateOptions{})
	return
}

// ApplyDeployment 应用 deployment
func (d *Deployment) Apply(filepath string) (deploy *appsv1.Deployment, err error) {
	deploy, err = d.Create(filepath)
	if errors.IsAlreadyExists(err) { // if deployment already exist, update it.
		deploy, err = d.Update(filepath)
	}
	return
}

// DeleteDeployment 删除 deployment
// 删除一个不存在的 deployment 会报错
func (d *Deployment) Delete(name string, forceDelete bool) (err error) {
	var gracePeriodSeconds int64
	if forceDelete {
		gracePeriodSeconds = 0
		err = d.client.AppsV1().Deployments(d.namespace).Delete(d.ctx,
			name, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds})
		return
	}
	err = d.client.AppsV1().Deployments(d.namespace).Delete(d.ctx, name, metav1.DeleteOptions{})
	return
}

// GetDeployment: 获取一个 appsv1.Deployment 对象
func (d *Deployment) Get(name string) (deploy *appsv1.Deployment, err error) {
	deploy, err = d.client.AppsV1().Deployments(d.namespace).Get(d.ctx, name, metav1.GetOptions{})
	return
}

// ListDeployments 获取一个 deployment 列表, 通过标签选择器来选择
func (d *Deployment) List(labelSelector string) (deployList *appsv1.DeploymentList, err error) {
	deployList, err = d.client.AppsV1().Deployments(d.namespace).List(d.ctx,
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &d.timeout, Limit: d.limit})
	return
}

// WatchDeployments 监听指定 deployment 的变化, 并调用 watchHandler 函数
func (d *Deployment) Watch(labelSelector string, addFunc, modifyFunc, deleteFunc func()) (err error) {
	var (
		watcher    watch.Interface
		timeout    = int64(0)
		isExist    bool
		deployList *appsv1.DeploymentList
	)

	// if event channel is closed, it means the server has closed the connection,
	// reconnect to kube-apiserver.
	for {
		//watcher, err := clientset.AppsV1().Deployments(namespace).Watch(ctx,
		//    metav1.SingleObject(metav1.ObjectMeta{Name: "dep", Namespace: namespace}))
		// 这个 timeout 一定要设置为 0, 否则 watcher 就会中断
		if watcher, err = d.client.AppsV1().Deployments(d.namespace).Watch(d.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout, Limit: d.limit}); err != nil {
			logrus.Error(err)
			return
		}
		if deployList, err = d.List(labelSelector); err != nil {
			logrus.Error(err)
			return
		}
		if len(deployList.Items) == 0 {
			isExist = false // deployment not exist
		} else {
			isExist = true // deployment exist
		}
		for {
			// kubernetes retains the resource event history, which includes this
			// initial event, so that when our program first start, we are automatically
			// notified of the deployment existence and current state.
			event, isOpen := <-watcher.ResultChan()

			if isOpen {
				switch event.Type {
				case watch.Added:
					// if deployment exist, skip deployment history add event.
					if !isExist {
						addFunc()
					}
					isExist = true
				case watch.Modified:
					modifyFunc()
					isExist = true
				case watch.Deleted:
					deleteFunc()
					isExist = false
				//case watch.Bookmark:
				//    logrus.Info("bookmark")
				//case watch.Error:
				//    logrus.Error("error")
				default: // do nothing
				}
			} else {
				// If event channel is closed, it means the server has closed the connection
				logrus.Info("reconnect to kube api-server.")
				break
			}
		}
	}
}
