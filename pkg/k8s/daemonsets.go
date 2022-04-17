package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type DaemonSet struct {
	kubeconfig string
	namespace  string

	ctx             context.Context
	config          *rest.Config
	restClient      *rest.RESTClient
	clientset       *kubernetes.Clientset
	dynamicClient   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient
	informerFactory informers.SharedInformerFactory
	informer        cache.SharedIndexInformer

	Options *HandlerOptions

	sync.Mutex
}

// NewDeployment new a daemonset handler from kubeconfig or in-cluster config
func NewDaemonSet(ctx context.Context, namespace, kubeconfig string) (daemonset *DaemonSet, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	daemonset = &DaemonSet{}

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
	// create a sharedInformerFactory for all namespaces.
	informerFactory = informers.NewSharedInformerFactory(clientset, time.Minute)

	if len(namespace) == 0 {
		namespace = metav1.NamespaceDefault
	}
	daemonset.kubeconfig = kubeconfig
	daemonset.namespace = namespace
	daemonset.ctx = ctx
	daemonset.config = config
	daemonset.restClient = restClient
	daemonset.clientset = clientset
	daemonset.dynamicClient = dynamicClient
	daemonset.discoveryClient = discoveryClient
	daemonset.informerFactory = informerFactory
	daemonset.informer = informerFactory.Apps().V1().DaemonSets().Informer()
	daemonset.Options = &HandlerOptions{}

	return
}
func (d *DaemonSet) Namespace() string {
	return d.namespace
}
func (in *DaemonSet) DeepCopy() *DaemonSet {
	out := new(DaemonSet)

	out.kubeconfig = in.kubeconfig
	out.namespace = in.namespace

	out.ctx = in.ctx
	out.config = in.config
	out.restClient = in.restClient
	out.clientset = in.clientset
	out.dynamicClient = in.dynamicClient
	out.discoveryClient = in.discoveryClient
	out.informerFactory = in.informerFactory
	out.informer = in.informer

	out.Options = &HandlerOptions{}
	out.Options.ListOptions = *in.Options.ListOptions.DeepCopy()
	out.Options.GetOptions = *in.Options.GetOptions.DeepCopy()
	out.Options.CreateOptions = *in.Options.CreateOptions.DeepCopy()
	out.Options.UpdateOptions = *in.Options.UpdateOptions.DeepCopy()
	out.Options.PatchOptions = *in.Options.PatchOptions.DeepCopy()
	out.Options.ApplyOptions = *in.Options.ApplyOptions.DeepCopy()

	return out
}
func (d *DaemonSet) setNamespace(namespace string) {
	d.Lock()
	defer d.Unlock()
	d.namespace = namespace
}

func (d *DaemonSet) WithNamespace(namespace string) *DaemonSet {
	ds := d.DeepCopy()
	ds.setNamespace(namespace)
	return ds
}
func (d *DaemonSet) WithDryRun() *DaemonSet {
	ds := d.DeepCopy()
	ds.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	ds.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	ds.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	ds.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	ds.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return ds
}
func (d *DaemonSet) SetTimeout(timeout int64) {
	d.Lock()
	defer d.Unlock()
	d.Options.ListOptions.TimeoutSeconds = &timeout
}
func (d *DaemonSet) SetLimit(limit int64) {
	d.Lock()
	defer d.Unlock()
	d.Options.ListOptions.Limit = limit
}
func (d *DaemonSet) SetForceDelete(force bool) {
	d.Lock()
	defer d.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		d.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		d.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create daemonset from map[string]interface{}
func (d *DaemonSet) CreateFromRaw(raw map[string]interface{}) (*appsv1.DaemonSet, error) {
	daemonset := &appsv1.DaemonSet{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, daemonset)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(daemonset.Namespace) != 0 {
		namespace = daemonset.Namespace
	} else {
		namespace = d.namespace
	}

	return d.clientset.AppsV1().DaemonSets(namespace).Create(d.ctx, daemonset, d.Options.CreateOptions)
}

// CreateFromBytes create daemonset from bytes
func (d *DaemonSet) CreateFromBytes(data []byte) (*appsv1.DaemonSet, error) {
	dsJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	daemonset := &appsv1.DaemonSet{}
	if err = json.Unmarshal(dsJson, daemonset); err != nil {
		return nil, err
	}

	var namespace string
	if len(daemonset.Namespace) != 0 {
		namespace = daemonset.Namespace
	} else {
		namespace = d.namespace
	}

	return d.clientset.AppsV1().DaemonSets(namespace).Create(d.ctx, daemonset, d.Options.CreateOptions)
}

// CreateFromFile create daemonset from yaml file
func (d *DaemonSet) CreateFromFile(path string) (*appsv1.DaemonSet, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return d.CreateFromBytes(data)
}

// Create create daemonset from yaml file, alias to "CreateFromFile"
func (d *DaemonSet) Create(path string) (*appsv1.DaemonSet, error) {
	return d.CreateFromFile(path)
}

// UpdateFromRaw update daemonset from map[string]interface{}
func (d *DaemonSet) UpdateFromRaw(raw map[string]interface{}) (*appsv1.DaemonSet, error) {
	daemonset := &appsv1.DaemonSet{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, daemonset)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(daemonset.Namespace) != 0 {
		namespace = daemonset.Namespace
	} else {
		namespace = d.namespace
	}

	return d.clientset.AppsV1().DaemonSets(namespace).Update(d.ctx, daemonset, d.Options.UpdateOptions)
}

// UpdateFromBytes update daemonset from bytes
func (d *DaemonSet) UpdateFromBytes(data []byte) (*appsv1.DaemonSet, error) {
	dsJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	daemonset := &appsv1.DaemonSet{}
	err = json.Unmarshal(dsJson, daemonset)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(daemonset.Namespace) != 0 {
		namespace = daemonset.Namespace
	} else {
		namespace = d.namespace
	}

	return d.clientset.AppsV1().DaemonSets(namespace).Update(d.ctx, daemonset, d.Options.UpdateOptions)
}

// UpdateFromFile update daemonset from yaml file
func (d *DaemonSet) UpdateFromFile(path string) (*appsv1.DaemonSet, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return d.UpdateFromBytes(data)
}

// Update update daemonset from file, alias to "UpdateFromFile"
func (d *DaemonSet) Update(path string) (*appsv1.DaemonSet, error) {
	return d.UpdateFromFile(path)
}

// ApplyFromRaw apply daemonset from map[string]interface{}
func (d *DaemonSet) ApplyFromRaw(raw map[string]interface{}) (*appsv1.DaemonSet, error) {
	pod := &appsv1.DaemonSet{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, pod)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(pod.Namespace) != 0 {
		namespace = pod.Namespace
	} else {
		namespace = d.namespace
	}

	pod, err = d.clientset.AppsV1().DaemonSets(namespace).Create(d.ctx, pod, d.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		pod, err = d.clientset.AppsV1().DaemonSets(namespace).Update(d.ctx, pod, d.Options.UpdateOptions)
	}
	return pod, err
}

// ApplyFromBytes apply daemonset from bytes
func (d *DaemonSet) ApplyFromBytes(data []byte) (daemonset *appsv1.DaemonSet, err error) {
	daemonset, err = d.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		daemonset, err = d.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply daemonset from yaml file
func (d *DaemonSet) ApplyFromFile(path string) (daemonset *appsv1.DaemonSet, err error) {
	daemonset, err = d.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		daemonset, err = d.UpdateFromFile(path)
	}
	return
}

// Apply apply daemonset from yaml file, alias to "ApplyFromFile"
func (d *DaemonSet) Apply(path string) (*appsv1.DaemonSet, error) {
	return d.ApplyFromFile(path)
}

// DeleteFromBytes delete daemonset from bytes
func (d *DaemonSet) DeleteFromBytes(data []byte) error {
	dsJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	daemonset := &appsv1.DaemonSet{}
	err = json.Unmarshal(dsJson, daemonset)
	if err != nil {
		return err
	}

	var namespace string
	if len(daemonset.Namespace) != 0 {
		namespace = daemonset.Namespace
	} else {
		namespace = d.namespace
	}

	return d.WithNamespace(namespace).DeleteByName(daemonset.Name)
}

// DeleteFromFile delete daemonset from yaml file
func (d *DaemonSet) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return d.DeleteFromBytes(data)
}

// DeleteByName delete daemonset by name
func (d *DaemonSet) DeleteByName(name string) error {
	return d.clientset.AppsV1().DaemonSets(d.namespace).Delete(d.ctx, name, d.Options.DeleteOptions)
}

// Delete delete daemonset by name, alias to "DeleteByName"
func (d *DaemonSet) Delete(name string) error {
	return d.DeleteByName(name)
}

// GetFromBytes get daemonset from bytes
func (d *DaemonSet) GetFromBytes(data []byte) (*appsv1.DaemonSet, error) {
	dsJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	daemonset := &appsv1.DaemonSet{}
	if err = json.Unmarshal(dsJson, daemonset); err != nil {
		return nil, err
	}

	var namespace string
	if len(daemonset.Namespace) != 0 {
		namespace = daemonset.Namespace
	} else {
		namespace = d.namespace
	}

	return d.WithNamespace(namespace).GetByName(daemonset.Name)
}

// GetFromFile get daemonset from yaml file
func (d *DaemonSet) GetFromFile(path string) (*appsv1.DaemonSet, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return d.GetFromBytes(data)
}

// GetByName get daemonset by name
func (d *DaemonSet) GetByName(name string) (*appsv1.DaemonSet, error) {
	return d.clientset.AppsV1().DaemonSets(d.namespace).Get(d.ctx, name, d.Options.GetOptions)
}

// Get get daemonset by name, alias to "GetByName"
func (d *DaemonSet) Get(name string) (*appsv1.DaemonSet, error) {
	return d.GetByName(name)
}

// ListByLabel list daemonsets by labels
func (d *DaemonSet) ListByLabel(labels string) (*appsv1.DaemonSetList, error) {
	listOptions := d.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return d.clientset.AppsV1().DaemonSets(d.namespace).List(d.ctx, *listOptions)
}

// List list daemonsets by labels, alias to "ListByLabel"
func (d *DaemonSet) List(labels string) (*appsv1.DaemonSetList, error) {
	return d.ListByLabel(labels)
}

// ListByNamespace list daemonsets by namespace
func (d *DaemonSet) ListByNamespace(namespace string) (*appsv1.DaemonSetList, error) {
	return d.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all daemonsets  in the k8s cluster
func (d *DaemonSet) ListAll() (*appsv1.DaemonSetList, error) {
	return d.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// GetPods get daemonset all pods
func (d *DaemonSet) GetPods(name string) (podList []string, err error) {
	// 检查 daemonset 是否就绪
	err = d.WaitReady(name, true)
	if err != nil {
		return
	}
	if !d.IsReady(name) {
		err = fmt.Errorf("daemonset %s not ready", name)
		return
	}

	// 创建一个 appsv1.DaemonSet 对象
	daemonset, err := d.Get(name)
	if err != nil {
		return
	}
	// 通过 spec.selector.matchLabels 找到 daemonset 创建的 pod
	matchLabels := daemonset.Spec.Selector.MatchLabels
	labelSelector := ""
	for key, value := range matchLabels {
		labelSelector = labelSelector + fmt.Sprintf("%s=%s,", key, value)
	}
	labelSelector = strings.TrimRight(labelSelector, ",")
	podObjList, err := d.clientset.CoreV1().Pods(d.namespace).List(d.ctx,
		metav1.ListOptions{LabelSelector: labelSelector})
	for _, pod := range podObjList.Items {
		podList = append(podList, pod.Name)
	}
	return
}

// GetPV get daemonset pv by name
func (d *DaemonSet) GetPV(name string) (pvList []string, err error) {
	var (
		pvcHandler *PersistentVolumeClaim
		pvcObj     *corev1.PersistentVolumeClaim
		pvcList    []string
	)
	err = d.WaitReady(name, true)
	if err != nil {
		return
	}
	if !d.IsReady(name) {
		err = fmt.Errorf("daemonset %s not ready", name)
		return
	}

	pvcHandler, err = NewPersistentVolumeClaim(d.ctx, d.namespace, d.kubeconfig)
	if err != nil {
		return
	}
	pvcList, err = d.GetPVC(name)
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

// GetPVC get daemonset pvc by name
func (d *DaemonSet) GetPVC(name string) (pvcList []string, err error) {
	err = d.WaitReady(name, true)
	if err != nil {
		return
	}
	if !d.IsReady(name) {
		err = fmt.Errorf("daemonset %s not ready", name)
		return
	}
	daemonset, err := d.Get(name)
	if err != nil {
		return
	}
	// 如果 volume.PersistentVolumeClaim 为 nil, 就不能再操作 volume.PersistentVolumeClaim.ClaimName
	for _, volume := range daemonset.Spec.Template.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			pvcList = append(pvcList, volume.PersistentVolumeClaim.ClaimName)
		}
	}
	return
}

// IsReady check if the daemonset is ready
func (d *DaemonSet) IsReady(name string) bool {
	daemonset, err := d.Get(name)
	if err != nil {
		return false
	}
	//log.Debug(daemonset.Status.DesiredNumberScheduled)
	//log.Debug(daemonset.Status.CurrentNumberScheduled)
	//log.Debug(daemonset.Status.NumberAvailable)
	//log.Debug(daemonset.Status.NumberReady)
	desiredNumberScheduled := daemonset.Status.DesiredNumberScheduled
	if daemonset.Status.CurrentNumberScheduled == desiredNumberScheduled &&
		daemonset.Status.NumberAvailable == desiredNumberScheduled &&
		daemonset.Status.NumberReady == desiredNumberScheduled {
		return true
	}
	return false
}

// WaitReady wait the daemonset to be th ready status
func (d *DaemonSet) WaitReady(name string, check bool) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
	)
	// 如果 daemonset 已经就绪, 就没必要继续 watch 了
	if d.IsReady(name) {
		return
	}
	// 是否判断 daemonset 是否存在
	if check {
		if _, err = d.Get(name); err != nil {
			return
		}
	}
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: d.namespace})
		listOptions.TimeoutSeconds = &timeout
		watcher, err = d.clientset.AppsV1().DaemonSets(d.namespace).Watch(d.ctx, listOptions)
		if err != nil {
			return
		}
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Modified:
				if d.IsReady(name) {
					watcher.Stop()
					return
				}
			case watch.Deleted:
				watcher.Stop()
				return fmt.Errorf("%s deleted", name)
			case watch.Bookmark:
				log.Debug("watch daemonset: bookmark.")
			case watch.Error:
				log.Debug("watch daemonset: error")

			}
		}
		log.Debug("watch daemonset: reconnect to kubernetes")
		watcher.Stop()
	}
}

// WatchByName watch daemonsets by name
func (d *DaemonSet) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: d.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = d.clientset.AppsV1().DaemonSets(d.namespace).Watch(d.ctx, listOptions); err != nil {
			return
		}
		if _, err = d.Get(name); err != nil {
			isExist = false // daemonsets not exist
		} else {
			isExist = true // daemonsets exist
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
				log.Debug("watch daemonset: bookmark")
			case watch.Error:
				log.Debug("watch daemonset: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch daemonset: reconnect to kubernetes")
		watcher.Stop()
	}
}

// WatchByLabel watch daemonsets by labelSelector
func (d *DaemonSet) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher       watch.Interface
		daemonsetList *appsv1.DaemonSetList
		timeout       = int64(0)
		isExist       bool
	)
	for {
		if watcher, err = d.clientset.AppsV1().DaemonSets(d.namespace).Watch(d.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if daemonsetList, err = d.List(labelSelector); err != nil {
			return
		}
		if len(daemonsetList.Items) == 0 {
			isExist = false // daemonset not exist
		} else {
			isExist = true // daemonset exist
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
				log.Debug("watch daemonset: bookmark")
			case watch.Error:
				log.Debug("watch daemonset: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch daemonset: reconnect to kubernetes")
		watcher.Stop()
	}
}

// Watch watch daemonsets by name, alias to "WatchByName"
func (d *DaemonSet) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return d.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}

// RunInformer
func (d *DaemonSet) RunInformer(
	addFunc func(obj interface{}),
	updateFunc func(oldObj, newObj interface{}),
	deleteFunc func(obj interface{}),
	stopCh chan struct{}) {
	d.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    addFunc,
		UpdateFunc: updateFunc,
		DeleteFunc: deleteFunc,
	})
	d.informer.Run(stopCh)
}
