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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type StatefulSet struct {
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

// new a StatefulSet handler from kubeconfig or in-cluster config
func NewStatefulSet(ctx context.Context, namespace, kubeconfig string) (statefulset *StatefulSet, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
	)
	statefulset = &StatefulSet{}

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
	statefulset.kubeconfig = kubeconfig
	statefulset.namespace = namespace

	statefulset.ctx = ctx
	statefulset.config = config
	statefulset.restClient = restClient
	statefulset.clientset = clientset
	statefulset.dynamicClient = dynamicClient
	statefulset.discoveryClient = discoveryClient

	statefulset.Options = &HandlerOptions{}

	return
}
func (s *StatefulSet) Namespace() string {
	return s.namespace
}
func (in *StatefulSet) DeepCopy() *StatefulSet {
	out := new(StatefulSet)

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
func (s *StatefulSet) setNamespace(namespace string) {
	s.Lock()
	defer s.Unlock()
	s.namespace = namespace
}
func (s *StatefulSet) WithNamespace(namespace string) *StatefulSet {
	sts := s.DeepCopy()
	sts.setNamespace(namespace)
	return sts
}
func (s *StatefulSet) WithDryRun() *StatefulSet {
	sts := s.DeepCopy()
	sts.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	sts.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	sts.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	sts.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	sts.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return sts
}
func (s *StatefulSet) SetTimeout(timeout int64) {
	s.Lock()
	defer s.Unlock()
	s.Options.ListOptions.TimeoutSeconds = &timeout
}
func (s *StatefulSet) SetLimit(limit int64) {
	s.Lock()
	defer s.Unlock()
	s.Options.ListOptions.Limit = limit
}
func (s *StatefulSet) SetForceDelete(force bool) {
	s.Lock()
	defer s.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		s.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		s.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create statefulset from map[string]interface{}
func (s *StatefulSet) CreateFromRaw(raw map[string]interface{}) (*appsv1.StatefulSet, error) {
	sts := &appsv1.StatefulSet{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, sts)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(sts.Namespace) != 0 {
		namespace = sts.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.AppsV1().StatefulSets(namespace).Create(s.ctx, sts, s.Options.CreateOptions)
}

// CreateFromBytes create statefulset from bytes
func (s *StatefulSet) CreateFromBytes(data []byte) (*appsv1.StatefulSet, error) {
	stsJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	sts := &appsv1.StatefulSet{}
	err = json.Unmarshal(stsJson, sts)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(sts.Namespace) != 0 {
		namespace = sts.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.AppsV1().StatefulSets(namespace).Create(s.ctx, sts, s.Options.CreateOptions)
}

// CreateFromFile create statefulset from yaml file
func (s *StatefulSet) CreateFromFile(path string) (*appsv1.StatefulSet, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.CreateFromBytes(data)
}

// Create create statefulset from yaml file, alias to "CreateFromFile"
func (s *StatefulSet) Create(path string) (*appsv1.StatefulSet, error) {
	return s.CreateFromFile(path)
}

// UpdateFromRaw update statefulset from map[string]interface{}
func (s *StatefulSet) UpdateFromRaw(raw map[string]interface{}) (*appsv1.StatefulSet, error) {
	sts := &appsv1.StatefulSet{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, sts)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(sts.Namespace) != 0 {
		namespace = sts.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.AppsV1().StatefulSets(namespace).Update(s.ctx, sts, s.Options.UpdateOptions)
}

// UpdateFromBytes update statefulset from bytes
func (s *StatefulSet) UpdateFromBytes(data []byte) (*appsv1.StatefulSet, error) {
	stsJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	sts := &appsv1.StatefulSet{}
	err = json.Unmarshal(stsJson, sts)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(sts.Namespace) != 0 {
		namespace = sts.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.AppsV1().StatefulSets(namespace).Update(s.ctx, sts, s.Options.UpdateOptions)
}

// UpdateFromFile update statefulset from yaml file
func (s *StatefulSet) UpdateFromFile(path string) (*appsv1.StatefulSet, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.UpdateFromBytes(data)
}

// Update update statefulset from file, alias to "UpdateFromFile"
func (s *StatefulSet) Update(path string) (*appsv1.StatefulSet, error) {
	return s.UpdateFromFile(path)
}

// ApplyFromRaw apply statefulset from map[string]interface{}
func (s *StatefulSet) ApplyFromRaw(raw map[string]interface{}) (*appsv1.StatefulSet, error) {
	sts := &appsv1.StatefulSet{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, sts)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(sts.Namespace) != 0 {
		namespace = sts.Namespace
	} else {
		namespace = s.namespace
	}

	sts, err = s.clientset.AppsV1().StatefulSets(namespace).Create(s.ctx, sts, s.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		sts, err = s.clientset.AppsV1().StatefulSets(namespace).Update(s.ctx, sts, s.Options.UpdateOptions)
	}
	return sts, err
}

// ApplyFromBytes apply statefulset from bytes
func (s *StatefulSet) ApplyFromBytes(data []byte) (statefulset *appsv1.StatefulSet, err error) {
	statefulset, err = s.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		statefulset, err = s.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply statefulset from yaml file
func (s *StatefulSet) ApplyFromFile(path string) (statefulset *appsv1.StatefulSet, err error) {
	statefulset, err = s.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		statefulset, err = s.UpdateFromFile(path)
	}
	return
}

// Apply apply statefulset from file, alias to "ApplyFromFile"
func (s *StatefulSet) Apply(path string) (*appsv1.StatefulSet, error) {
	return s.ApplyFromFile(path)
}

// DeleteFromBytes delete statefulset from bytes
func (s *StatefulSet) DeleteFromBytes(data []byte) error {
	stsJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	sts := &appsv1.StatefulSet{}
	err = json.Unmarshal(stsJson, sts)
	if err != nil {
		return err
	}

	var namespace string
	if len(sts.Namespace) != 0 {
		namespace = sts.Namespace
	} else {
		namespace = s.namespace
	}

	return s.WithNamespace(namespace).DeleteByName(sts.Name)
}

// DeleteFromFile delete statefulset from yaml file
func (s *StatefulSet) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return s.DeleteFromBytes(data)
}

// DeleteByName delete statefulset by name
func (s *StatefulSet) DeleteByName(name string) error {
	return s.clientset.AppsV1().StatefulSets(s.namespace).Delete(s.ctx, name, s.Options.DeleteOptions)
}

// Delete delete statefulset by name, alias to "DeleteByName"
func (s *StatefulSet) Delete(name string) error {
	return s.DeleteByName(name)
}

// GetFromBytes get statefulset from bytes
func (s *StatefulSet) GetFromBytes(data []byte) (*appsv1.StatefulSet, error) {
	stsJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	sts := &appsv1.StatefulSet{}
	err = json.Unmarshal(stsJson, sts)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(sts.Namespace) != 0 {
		namespace = sts.Namespace
	} else {
		namespace = s.namespace
	}

	return s.WithNamespace(namespace).GetByName(sts.Name)
}

// GetFromFile get statefulset from file
func (s *StatefulSet) GetFromFile(path string) (*appsv1.StatefulSet, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.GetFromBytes(data)
}

// GetByName get statefulset by name
func (s *StatefulSet) GetByName(name string) (*appsv1.StatefulSet, error) {
	return s.clientset.AppsV1().StatefulSets(s.namespace).Get(s.ctx, name, s.Options.GetOptions)
}

// Get get statefulset by name, alias to "GetByName"
func (s *StatefulSet) Get(name string) (*appsv1.StatefulSet, error) {
	return s.GetByName(name)
}

// ListByLabel list statefulsets by labels
func (s *StatefulSet) ListByLabel(labels string) (*appsv1.StatefulSetList, error) {
	s.Options.ListOptions.LabelSelector = labels
	listOptions := s.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return s.clientset.AppsV1().StatefulSets(s.namespace).List(s.ctx, *listOptions)
}

// List list statefulsets by labels, alias to "ListByLabel
func (s *StatefulSet) List(labels string) (*appsv1.StatefulSetList, error) {
	return s.ListByLabel(labels)
}

// ListByNamespace list statefulsets by namespace
func (s *StatefulSet) ListByNamespace(namespace string) (*appsv1.StatefulSetList, error) {
	return s.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all statefulsets in the k8s cluster
func (s *StatefulSet) ListAll() (*appsv1.StatefulSetList, error) {
	return s.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// get statefulset all pod
func (s *StatefulSet) GetPods(name string) (podList []string, err error) {
	// 判断 statefulset 是否就绪
	err = s.WaitReady(name, true)
	if err != nil {
		return
	}
	if !s.IsReady(name) {
		err = fmt.Errorf("statefulset %s not ready", name)
		return
	}

	// 获取一个 *appsv1.Statefulset 对象
	statefulset, err := s.Get(name)
	if err != nil {
		return
	}
	// 通过 statefulset.spec.selector.matchLabels 来获取 statefulset 创建的 pod
	matchLabels := statefulset.Spec.Selector.MatchLabels
	labelSelector := ""
	for key, value := range matchLabels {
		labelSelector = labelSelector + fmt.Sprintf("%s=%s,", key, value)
	}
	labelSelector = strings.TrimRight(labelSelector, ",")
	podObjList, err := s.clientset.CoreV1().Pods(s.namespace).List(s.ctx,
		metav1.ListOptions{LabelSelector: labelSelector})
	// 获取所有 Pod, 并放入 podList 列表中
	for _, pod := range podObjList.Items {
		podList = append(podList, pod.Name)
	}
	return
}

// get statefulset pv by name
func (s *StatefulSet) GetPV(name string) (pvList []string, err error) {
	var (
		pvcHandler *PersistentVolumeClaim
		pvcObj     *corev1.PersistentVolumeClaim
		pvcList    []string
	)
	err = s.WaitReady(name, true)
	if err != nil {
		return
	}
	if !s.IsReady(name) {
		err = fmt.Errorf("statefulset %s not ready", name)
		return
	}
	pvcHandler, err = NewPersistentVolumeClaim(s.ctx, s.namespace, s.kubeconfig)
	if err != nil {
		return
	}
	pvcList, err = s.GetPVC(name)
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

// get statefulset pvc by name
func (s *StatefulSet) GetPVC(name string) (pvcList []string, err error) {
	// 等待 statefulset 就绪
	err = s.WaitReady(name, true)
	if err != nil {
		return
	}
	// 再次判断 statefulset 是否就绪
	if !s.IsReady(name) {
		err = fmt.Errorf("statefulset %s not ready", name)
		return
	}
	statefulset, err := s.Get(name)
	if err != nil {
		return
	}
	// pvc 的格式为 statefulset name + VolumeClaimTemplates name + replicas 编号
	for _, pvc := range statefulset.Spec.VolumeClaimTemplates {
		for i := int32(0); i < *statefulset.Spec.Replicas; i++ {
			pvcList = append(pvcList, fmt.Sprintf("%s-%s-%d",
				pvc.ObjectMeta.Name, statefulset.Name, i))
		}
	}
	return
}

// check if the statefulset is ready
func (s *StatefulSet) IsReady(name string) bool {
	statefulset, err := s.Get(name)
	if err != nil {
		return false
	}
	// 如果 statefulset 的 replicaas 等于 status.AvailableReplicas 的个数
	// 就表明 statefulset 的所有 pod 都就绪了.
	if *statefulset.Spec.Replicas == statefulset.Status.AvailableReplicas {
		return true
	}

	return false
}

// wait the statefulset to be in the ready status
func (s *StatefulSet) WaitReady(name string, check bool) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
	)
	// watch 之前先判断 statefulset 是否就绪, 如果已经继续就没必要继续 watch 了
	if s.IsReady(name) {
		return
	}
	// 是否判断 statefulset 是否存在
	if check {
		if _, err = s.Get(name); err != nil {
			return
		}
	}
	listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: s.namespace})
	listOptions.TimeoutSeconds = &timeout
	watcher, err = s.clientset.AppsV1().StatefulSets(s.namespace).Watch(s.ctx, listOptions)
	if err != nil {
		return
	}
	// 由于 watcher 会因为 keepalive 超时被 kube-apiserver 中断, 所以需要循环创建 watcher
	for {
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Modified:
				if s.IsReady(name) {
					watcher.Stop()
					return
				}
			case watch.Deleted:
				watcher.Stop()
				return fmt.Errorf("%s deleted", name)
			case watch.Bookmark:
				log.Debug("watch statefulset: bookmark")
			case watch.Error:
				log.Debug("watch statefulset: error")
			}
		}
		log.Debug("watch statefulset: reconnect to kubernetes")
		watcher.Stop()
	}
}

// watch statefulset by name
func (s *StatefulSet) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: s.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = s.clientset.AppsV1().StatefulSets(s.namespace).Watch(s.ctx, listOptions); err != nil {
			return
		}
		if _, err = s.Get(name); err != nil {
			isExist = false // statefulset not exist
		} else {
			isExist = true // statefulset exist
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
				log.Debug("watch statefulset: bookmark")
			case watch.Error:
				log.Debug("watch statefulset: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch statefulset: reconnect to kubernetes")
		watcher.Stop()
	}
}

// watch statefulset by labelSelector
func (s *StatefulSet) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher         watch.Interface
		statefulsetList *appsv1.StatefulSetList
		timeout         = int64(0)
		isExist         bool
	)
	for {
		if watcher, err = s.clientset.AppsV1().StatefulSets(s.namespace).Watch(s.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if statefulsetList, err = s.List(labelSelector); err != nil {
			return
		}
		if len(statefulsetList.Items) == 0 {
			isExist = false // statefulset not exist
		} else {
			isExist = true // statefulset exist
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
				log.Debug("watch statefulset: bookmark")
			case watch.Error:
				log.Debug("watch statefulset: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch statefulset: reconnect to kubernetes")
		watcher.Stop()
	}
}

// watch statefulset by name, alias to "WatchByName"
func (s *StatefulSet) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return s.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
