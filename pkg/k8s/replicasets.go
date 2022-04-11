package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ReplicaSet struct {
	kubeconfig string
	namespace  string

	ctx             context.Context
	config          *rest.Config
	restClient      *rest.RESTClient
	clientset       *kubernetes.Clientset
	dynamicClient   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient

	Options *HandlerOptions

	sync.Mutex
}

// new a replicaset handler from kubeconfig or in-cluster config
func NewReplicaSet(ctx context.Context, namespace, kubeconfig string) (replicaset *ReplicaSet, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
	)
	replicaset = &ReplicaSet{}

	// create rest config
	if len(kubeconfig) != 0 {
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return
		}
	} else {
		// create the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return
		}
	}

	// setup APIPath, GroupVersion and NegotiatedSerializer before initializing a RESTClient
	config.APIPath = "api"
	config.GroupVersion = &appsv1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs
	// create a RESTClient for the given config
	restClient, err = rest.RESTClientFor(config)
	if err != nil {
		return
	}
	// create a Clientset for the given config
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return
	}
	// create a dynamic client for the given config
	dynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return
	}
	// create a DiscoveryClient for the given config
	discoveryClient, err = discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return
	}

	if len(namespace) == 0 {
		namespace = metav1.NamespaceDefault
	}
	replicaset.kubeconfig = kubeconfig
	replicaset.namespace = namespace

	replicaset.ctx = ctx
	replicaset.config = config
	replicaset.restClient = restClient
	replicaset.clientset = clientset
	replicaset.dynamicClient = dynamicClient
	replicaset.discoveryClient = discoveryClient

	replicaset.Options = &HandlerOptions{}

	return
}
func (r *ReplicaSet) Namespace() string {
	return r.namespace
}
func (in *ReplicaSet) DeepCopy() *ReplicaSet {
	out := new(ReplicaSet)

	out.kubeconfig = in.kubeconfig
	out.namespace = in.namespace

	out.ctx = in.ctx
	out.config = in.config
	out.restClient = in.restClient
	out.clientset = in.clientset
	out.dynamicClient = in.dynamicClient
	out.discoveryClient = in.discoveryClient

	out.Options = &HandlerOptions{}
	out.Options.ListOptions = *in.Options.ListOptions.DeepCopy()
	out.Options.GetOptions = *in.Options.GetOptions.DeepCopy()
	out.Options.CreateOptions = *in.Options.CreateOptions.DeepCopy()
	out.Options.UpdateOptions = *in.Options.UpdateOptions.DeepCopy()
	out.Options.PatchOptions = *in.Options.PatchOptions.DeepCopy()
	out.Options.ApplyOptions = *in.Options.ApplyOptions.DeepCopy()

	return out
}
func (r *ReplicaSet) setNamespace(namespace string) {
	r.Lock()
	defer r.Unlock()
	r.namespace = namespace
}

func (r *ReplicaSet) WithNamespace(namespace string) *ReplicaSet {
	rs := r.DeepCopy()
	rs.setNamespace(namespace)
	return rs
}
func (r *ReplicaSet) WithDryRun() *ReplicaSet {
	rs := r.DeepCopy()
	rs.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	rs.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	rs.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	rs.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	rs.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return rs
}
func (r *ReplicaSet) SetTimeout(timeout int64) {
	r.Lock()
	defer r.Unlock()
	r.Options.ListOptions.TimeoutSeconds = &timeout
}
func (r *ReplicaSet) SetLimit(limit int64) {
	r.Lock()
	defer r.Unlock()
	r.Options.ListOptions.Limit = limit
}
func (r *ReplicaSet) SetForceDelete(force bool) {
	r.Lock()
	defer r.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		r.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		r.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// create replicaset from bytes
func (r *ReplicaSet) CreateFromBytes(data []byte) (*appsv1.ReplicaSet, error) {
	dsJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	replicaset := &appsv1.ReplicaSet{}
	if err = json.Unmarshal(dsJson, replicaset); err != nil {
		return nil, err
	}

	var namespace string
	if len(replicaset.Namespace) != 0 {
		namespace = replicaset.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.AppsV1().ReplicaSets(namespace).Create(r.ctx, replicaset, r.Options.CreateOptions)
}

// create replicaset from file
func (r *ReplicaSet) CreateFromFile(path string) (*appsv1.ReplicaSet, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.CreateFromBytes(data)
}

// create replicaset from file, alias to "CreateFromFile"
func (r *ReplicaSet) Create(path string) (*appsv1.ReplicaSet, error) {
	return r.CreateFromFile(path)
}

// update replicaset from bytes
func (r *ReplicaSet) UpdateFromBytes(data []byte) (*appsv1.ReplicaSet, error) {
	dsJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	replicaset := &appsv1.ReplicaSet{}
	err = json.Unmarshal(dsJson, replicaset)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(replicaset.Namespace) != 0 {
		namespace = replicaset.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.AppsV1().ReplicaSets(namespace).Update(r.ctx, replicaset, r.Options.UpdateOptions)
}

// update replicaset from file
func (r *ReplicaSet) UpdateFromFile(path string) (*appsv1.ReplicaSet, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.UpdateFromBytes(data)
}

// update replicaset from file, alias to "UpdateFromFile"
func (r *ReplicaSet) Update(path string) (*appsv1.ReplicaSet, error) {
	return r.UpdateFromFile(path)
}

// apply replicaset from bytes
func (r *ReplicaSet) ApplyFromBytes(data []byte) (replicaset *appsv1.ReplicaSet, err error) {
	replicaset, err = r.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		replicaset, err = r.UpdateFromBytes(data)
	}
	return
}

// apply replicaset from file
func (r *ReplicaSet) ApplyFromFile(path string) (replicaset *appsv1.ReplicaSet, err error) {
	replicaset, err = r.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		replicaset, err = r.UpdateFromFile(path)
	}
	return
}

// apply replicaset from file, alias to "ApplyFromFile"
func (r *ReplicaSet) Apply(path string) (*appsv1.ReplicaSet, error) {
	return r.ApplyFromFile(path)
}

// delete replicaset from bytes
func (r *ReplicaSet) DeleteFromBytes(data []byte) error {
	dsJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	replicaset := &appsv1.ReplicaSet{}
	err = json.Unmarshal(dsJson, replicaset)
	if err != nil {
		return err
	}

	var namespace string
	if len(replicaset.Namespace) != 0 {
		namespace = replicaset.Namespace
	} else {
		namespace = r.namespace
	}

	return r.WithNamespace(namespace).DeleteByName(replicaset.Name)
}

// delete replicaset from file
func (r *ReplicaSet) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return r.DeleteFromBytes(data)
}

// delete replicaset by name
func (r *ReplicaSet) DeleteByName(name string) error {
	return r.clientset.AppsV1().ReplicaSets(r.namespace).Delete(r.ctx, name, r.Options.DeleteOptions)
}

// delete replicaset by name, alias to "DeleteByName"
func (r *ReplicaSet) Delete(name string) error {
	return r.DeleteByName(name)
}

// list replicaset by labelSelector
func (r *ReplicaSet) List(labelSelector string) (*appsv1.ReplicaSetList, error) {
	r.Options.ListOptions.LabelSelector = labelSelector
	return r.clientset.AppsV1().ReplicaSets(r.namespace).List(r.ctx, r.Options.ListOptions)
}

// get replicaset from bytes
func (r *ReplicaSet) GetFromBytes(data []byte) (*appsv1.ReplicaSet, error) {
	dsJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	replicaset := &appsv1.ReplicaSet{}
	if err = json.Unmarshal(dsJson, replicaset); err != nil {
		return nil, err
	}

	var namespace string
	if len(replicaset.Namespace) != 0 {
		namespace = replicaset.Namespace
	} else {
		namespace = r.namespace
	}

	return r.WithNamespace(namespace).GetByName(replicaset.Name)
}

// get replicaset from file
func (r *ReplicaSet) GetFromFile(path string) (*appsv1.ReplicaSet, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return r.GetFromBytes(data)
}

// get replicaset by name
func (r *ReplicaSet) GetByName(name string) (*appsv1.ReplicaSet, error) {
	return r.clientset.AppsV1().ReplicaSets(r.namespace).Get(r.ctx, name, r.Options.GetOptions)
}

// get replicaset by name, alias to "GetByName"
func (r *ReplicaSet) Get(name string) (*appsv1.ReplicaSet, error) {
	return r.GetByName(name)
}

// get replicaset all pods
func (r *ReplicaSet) GetPods(name string) (podList []string, err error) {
	// 检查 replicaset 是否就绪
	err = r.WaitReady(name, true)
	if err != nil {
		return
	}
	if !r.IsReady(name) {
		err = fmt.Errorf("replicaset %s not ready", name)
		return
	}

	// 创建一个 appsv1.ReplicaSet 对象
	replicaset, err := r.Get(name)
	if err != nil {
		return
	}
	// 通过 spec.selector.matchLabels 找到 replicaset 创建的 pod
	matchLabels := replicaset.Spec.Selector.MatchLabels
	labelSelector := ""
	for key, value := range matchLabels {
		labelSelector = labelSelector + fmt.Sprintf("%s=%s,", key, value)
	}
	labelSelector = strings.TrimRight(labelSelector, ",")
	podObjList, err := r.clientset.CoreV1().Pods(r.namespace).List(r.ctx,
		metav1.ListOptions{LabelSelector: labelSelector})
	for _, pod := range podObjList.Items {
		podList = append(podList, pod.Name)
	}
	return
}

// get replicaset pv by name
func (r *ReplicaSet) GetPV(name string) (pvList []string, err error) {
	var (
		pvcHandler *PersistentVolumeClaim
		pvcObj     *corev1.PersistentVolumeClaim
		pvcList    []string
	)
	err = r.WaitReady(name, true)
	if err != nil {
		return
	}
	if !r.IsReady(name) {
		err = fmt.Errorf("replicaset %s not ready", name)
		return
	}

	pvcHandler, err = NewPersistentVolumeClaim(r.ctx, r.namespace, r.kubeconfig)
	if err != nil {
		return
	}
	pvcList, err = r.GetPVC(name)
	if err != nil {
		return
	}

	for _, pvcName := range pvcList {
		pvcObj, err = pvcHandler.Get(pvcName)
		if err != nil {
			return
		}
		pvList = append(pvList, pvcObj.Spec.VolumeName)
	}

	return
}

// get replicaset pvc by name
func (r *ReplicaSet) GetPVC(name string) (pvcList []string, err error) {
	err = r.WaitReady(name, true)
	if err != nil {
		return
	}
	if !r.IsReady(name) {
		err = fmt.Errorf("replicaset %s not ready", name)
		return
	}
	replicaset, err := r.Get(name)
	if err != nil {
		return
	}
	// 如果 volume.PersistentVolumeClaim 为 nil, 就不能再操作 volume.PersistentVolumeClaim.ClaimName
	for _, volume := range replicaset.Spec.Template.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			pvcList = append(pvcList, volume.PersistentVolumeClaim.ClaimName)
		}
	}
	return
}

// check if the replicaset is ready
func (r *ReplicaSet) IsReady(name string) bool {
	replicaset, err := r.Get(name)
	if err != nil {
		return false
	}
	replicas := replicaset.Status.Replicas
	if replicaset.Status.AvailableReplicas == replicas &&
		replicaset.Status.FullyLabeledReplicas == replicas &&
		replicaset.Status.ReadyReplicas == replicas {
		return true
	}
	return false
}

// wait the replicaset to be th ready status
func (r *ReplicaSet) WaitReady(name string, check bool) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
	)
	// 如果 replicaset 已经就绪, 就没必要继续 watch 了
	if r.IsReady(name) {
		return
	}
	// 是否判断 replicaset 是否存在
	if check {
		if _, err = r.Get(name); err != nil {
			return
		}
	}
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: r.namespace})
		listOptions.TimeoutSeconds = &timeout
		watcher, err = r.clientset.AppsV1().ReplicaSets(r.namespace).Watch(r.ctx, listOptions)
		if err != nil {
			return
		}
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Modified:
				if r.IsReady(name) {
					watcher.Stop()
					return
				}
			case watch.Deleted:
				watcher.Stop()
				return fmt.Errorf("%s deleted", name)
			case watch.Bookmark:
				log.Debug("watch replicaset: bookmark.")
			case watch.Error:
				log.Debug("watch replicaset: error")

			}
		}
		log.Debug("watch replicaset: reconnect to kubernetes")
		watcher.Stop()
	}
}

// watch replicasets by name
func (r *ReplicaSet) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: r.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = r.clientset.AppsV1().ReplicaSets(r.namespace).Watch(r.ctx, listOptions); err != nil {
			return
		}
		if _, err = r.Get(name); err != nil {
			isExist = false // replicasets not exist
		} else {
			isExist = true // replicasets exist
		}
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				if !isExist {
					addFunc(x)
				}
				isExist = true
			case watch.Modified:
				modifyFunc(x)
				isExist = true
			case watch.Deleted:
				deleteFunc(x)
				isExist = false
			case watch.Bookmark:
				log.Debug("watch replicaset: bookmark")
			case watch.Error:
				log.Debug("watch replicaset: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch replicaset: reconnect to kubernetes")
		watcher.Stop()
	}
}

// watch replicasets by labelSelector
func (r *ReplicaSet) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher        watch.Interface
		replicasetList *appsv1.ReplicaSetList
		timeout        = int64(0)
		isExist        bool
	)
	for {
		if watcher, err = r.clientset.AppsV1().ReplicaSets(r.namespace).Watch(r.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if replicasetList, err = r.List(labelSelector); err != nil {
			return
		}
		if len(replicasetList.Items) == 0 {
			isExist = false // replicaset not exist
		} else {
			isExist = true // replicaset exist
		}
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				if !isExist {
					addFunc(x)
				}
				isExist = true
			case watch.Modified:
				modifyFunc(x)
				isExist = true
			case watch.Deleted:
				deleteFunc(x)
				isExist = false
			case watch.Bookmark:
				log.Debug("watch replicaset: bookmark")
			case watch.Error:
				log.Debug("watch replicaset: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch replicaset: reconnect to kubernetes")
		watcher.Stop()
	}
}

// watch replicasets by name, alias to "WatchByName"
func (r *ReplicaSet) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return r.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
