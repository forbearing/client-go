package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
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

type ReplicationController struct {
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

// new a ReplicationController handler from kubeconfig or in-cluster config
func NewReplicationController(ctx context.Context, namespace, kubeconfig string) (rc *ReplicationController, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	rc = &ReplicationController{}

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

	// setup APIPath, GroupVersion and NegotiatedSerializer before initializing a RESTClient
	config.APIPath = "api"
	config.GroupVersion = &corev1.SchemeGroupVersion
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
	rc.kubeconfig = kubeconfig
	rc.namespace = namespace
	rc.ctx = ctx
	rc.config = config
	rc.restClient = restClient
	rc.clientset = clientset
	rc.dynamicClient = dynamicClient
	rc.discoveryClient = discoveryClient
	rc.informerFactory = informerFactory
	rc.informer = informerFactory.Core().V1().ReplicationControllers().Informer()
	rc.Options = &HandlerOptions{}

	return
}
func (r *ReplicationController) Namespace() string {
	return r.namespace
}
func (in *ReplicationController) DeepCopy() *ReplicationController {
	out := new(ReplicationController)

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

func (r *ReplicationController) setNamespace(namespace string) {
	r.Lock()
	defer r.Unlock()
	r.namespace = namespace
}
func (r *ReplicationController) WithNamespace(namespace string) *ReplicationController {
	rc := r.DeepCopy()
	rc.setNamespace(namespace)
	return rc
}
func (r *ReplicationController) WithDryRun() *ReplicationController {
	rc := r.DeepCopy()
	rc.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	rc.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	rc.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	rc.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	rc.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return rc
}
func (r *ReplicationController) SetLimit(limit int64) {
	r.Lock()
	defer r.Unlock()
	r.Options.ListOptions.Limit = limit
}
func (r *ReplicationController) SetTimeout(timeout int64) {
	r.Lock()
	defer r.Unlock()
	r.Options.ListOptions.TimeoutSeconds = &timeout
}
func (r *ReplicationController) SetForceDelete(force bool) {
	r.Lock()
	defer r.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		r.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		r.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create replicationcontroller from map[string]interface{}
func (r *ReplicationController) CreateFromRaw(raw map[string]interface{}) (*corev1.ReplicationController, error) {
	rc := &corev1.ReplicationController{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, rc)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(rc.Namespace) != 0 {
		namespace = rc.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.CoreV1().ReplicationControllers(namespace).Create(r.ctx, rc, r.Options.CreateOptions)
}

// CreateFromBytes create replicationcontroller from bytes
func (r *ReplicationController) CreateFromBytes(data []byte) (*corev1.ReplicationController, error) {
	rcJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	rc := &corev1.ReplicationController{}
	err = json.Unmarshal(rcJson, rc)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(rc.Namespace) != 0 {
		namespace = rc.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.CoreV1().ReplicationControllers(namespace).Create(r.ctx, rc, r.Options.CreateOptions)
}

// CreateFromFile create replicationcontroller from yaml file
func (r *ReplicationController) CreateFromFile(path string) (*corev1.ReplicationController, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.CreateFromBytes(data)
}

// Create create replicationcontroller from yaml file, alias to "CreateFromFile"
func (r *ReplicationController) Create(path string) (*corev1.ReplicationController, error) {
	return r.CreateFromFile(path)
}

// UpdateFromRaw update replicationcontroller from map[string]interface{}
func (r *ReplicationController) UpdateFromRaw(raw map[string]interface{}) (*corev1.ReplicationController, error) {
	rc := &corev1.ReplicationController{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, rc)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(rc.Namespace) != 0 {
		namespace = rc.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.CoreV1().ReplicationControllers(namespace).Update(r.ctx, rc, r.Options.UpdateOptions)
}

// UpdateFromBytes update replicationcontroller from bytes
func (r *ReplicationController) UpdateFromBytes(data []byte) (*corev1.ReplicationController, error) {
	rcJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	rc := &corev1.ReplicationController{}
	err = json.Unmarshal(rcJson, rc)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(rc.Namespace) != 0 {
		namespace = rc.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.CoreV1().ReplicationControllers(namespace).Update(r.ctx, rc, r.Options.UpdateOptions)
}

// UpdateFromFile update replicationcontroller from yaml file
func (r *ReplicationController) UpdateFromFile(path string) (*corev1.ReplicationController, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.UpdateFromBytes(data)
}

// Update update replicationcontroller from yaml file, alias to "UpdateFromFile"
func (r *ReplicationController) Update(path string) (*corev1.ReplicationController, error) {
	return r.UpdateFromFile(path)
}

// ApplyFromRaw apply replicationcontroller from map[string]interface{}
func (r *ReplicationController) ApplyFromRaw(raw map[string]interface{}) (*corev1.ReplicationController, error) {
	rc := &corev1.ReplicationController{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, rc)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(rc.Namespace) != 0 {
		namespace = rc.Namespace
	} else {
		namespace = r.namespace
	}

	rc, err = r.clientset.CoreV1().ReplicationControllers(namespace).Create(r.ctx, rc, r.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		rc, err = r.clientset.CoreV1().ReplicationControllers(namespace).Update(r.ctx, rc, r.Options.UpdateOptions)
	}
	return rc, err
}

// ApplyFromBytes apply replicationcontroller from bytes
func (r *ReplicationController) ApplyFromBytes(data []byte) (rc *corev1.ReplicationController, err error) {
	rc, err = r.CreateFromBytes(data)
	if k8serrors.IsAlreadyExists(err) {
		log.Debug(err)
		rc, err = r.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply replicationcontroller from yaml file
func (r *ReplicationController) ApplyFromFile(path string) (rc *corev1.ReplicationController, err error) {
	rc, err = r.CreateFromFile(path)
	if k8serrors.IsAlreadyExists(err) {
		rc, err = r.UpdateFromFile(path)
	}
	return
}

// Apply apply replicationcontroller from yaml file, alias to "ApplyFromFile"
func (r *ReplicationController) Apply(path string) (*corev1.ReplicationController, error) {
	return r.ApplyFromFile(path)
}

// DeleteFromBytes delete replicationcontroller from bytes
func (r *ReplicationController) DeleteFromBytes(data []byte) error {
	rcJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	rc := &corev1.ReplicationController{}
	err = json.Unmarshal(rcJson, rc)
	if err != nil {
		return err
	}

	var namespace string
	if len(rc.Namespace) != 0 {
		namespace = rc.Namespace
	} else {
		namespace = r.namespace
	}

	return r.WithNamespace(namespace).DeleteByName(rc.Name)
}

// DeleteFromFile delete replicationcontroller from yaml file
func (r *ReplicationController) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return r.DeleteFromBytes(data)
}

// DeleteByName delete replicationcontroller by name
func (r *ReplicationController) DeleteByName(name string) error {
	return r.clientset.CoreV1().ReplicationControllers(r.namespace).Delete(r.ctx, name, r.Options.DeleteOptions)
}

// Delete delete replicationcontroller by name, alias to "DeleteByName"
func (r *ReplicationController) Delete(name string) error {
	return r.DeleteByName(name)
}

// ListByLabel list replicationcontrollers by labels
func (r *ReplicationController) ListByLabel(labels string) (*corev1.ReplicationControllerList, error) {
	listOptions := r.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return r.clientset.CoreV1().ReplicationControllers(r.namespace).List(r.ctx, *listOptions)
}

// List list replicationcontrollers by labels, alias to "ListByLabel"
func (r *ReplicationController) List(labels string) (*corev1.ReplicationControllerList, error) {
	return r.ListByLabel(labels)
}

// ListByNamespace list replicationcontrollers by namespace
func (r *ReplicationController) ListByNamespace(namespace string) (*corev1.ReplicationControllerList, error) {
	return r.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all replicationcontrollers in the k8s cluster
func (r *ReplicationController) ListAll() (*corev1.ReplicationControllerList, error) {
	return r.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// GetFromBytes get replicationcontroller from bytes
func (r *ReplicationController) GetFromBytes(data []byte) (*corev1.ReplicationController, error) {
	rcJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	rc := &corev1.ReplicationController{}
	if err = json.Unmarshal(rcJson, rc); err != nil {
		return nil, err
	}

	var namespace string
	if len(rc.Namespace) != 0 {
		namespace = rc.Namespace
	} else {
		namespace = r.namespace
	}

	return r.WithNamespace(namespace).GetByName(rc.Name)
}

// GetFromFile get replicationcontroller from yaml file
func (r *ReplicationController) GetFromFile(path string) (*corev1.ReplicationController, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.GetFromBytes(data)
}

// GetByName get replicationcontroller by name
func (r *ReplicationController) GetByName(name string) (*corev1.ReplicationController, error) {
	return r.clientset.CoreV1().ReplicationControllers(r.namespace).Get(r.ctx, name, r.Options.GetOptions)
}

// Get get replicationcontroller by name
func (r *ReplicationController) Get(name string) (replicationcontroller *corev1.ReplicationController, err error) {
	return r.GetByName(name)
}

// IsReady check if the replicationcontroller is ready
func (r *ReplicationController) IsReady(name string) bool {
	// ?????? *corev1.ReplicationController ??????
	rc, err := r.Get(name)
	if err != nil {
		return false
	}
	replicas := rc.Status.Replicas
	if rc.Status.AvailableReplicas == replicas &&
		rc.Status.FullyLabeledReplicas == replicas &&
		rc.Status.ReadyReplicas == replicas {
		return true
	}
	return false
}

// WaitReady wait for the replicationcontroller to be in the ready status
func (r *ReplicationController) WaitReady(name string, check bool) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
	)
	// ??? watch ?????????????????? replicationcontroller ????????????, ??????????????????????????? watch ???
	if r.IsReady(name) {
		return
	}
	// ???????????? replicationcontroller ????????????
	if check {
		if _, err = r.Get(name); err != nil {
			return
		}
	}
	for {
		// replicationcontroller ????????????, ????????????????????? replicationcontroller ?????????
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: r.namespace})
		listOptions.TimeoutSeconds = &timeout
		watcher, err = r.clientset.CoreV1().ReplicationControllers(r.namespace).Watch(r.ctx, listOptions)
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
				log.Debug("watch replicationcontroller: bookmark")
			case watch.Error:
				log.Debug("watch replicationcontroller: error")
			}
		}
		// watcher ?????? keepalive ?????????????????????, ????????? channel
		log.Debug("watch replicationcontroller: reconnect to kubernetes")
		watcher.Stop()
	}
}

// WatchByName watch replicationcontrollers by labelSelector
func (r *ReplicationController) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: r.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = r.clientset.CoreV1().ReplicationControllers(r.namespace).Watch(r.ctx, listOptions); err != nil {
			return
		}
		if _, err = r.Get(name); err != nil {
			isExist = false // replicationcontroller not exist
		} else {
			isExist = true // replicationcontroller exist
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
				log.Debug("watch replicationcontroller: bookmark")
			case watch.Error:
				log.Debug("watch replicationcontroller: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch replicationcontroller: reconnect to kubernetes")
		watcher.Stop()
	}
}

// WatchByLabel watch replicationcontrollers by labelSelector
func (r *ReplicationController) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher                   watch.Interface
		replicationcontrollerList *corev1.ReplicationControllerList
		timeout                   = int64(0)
		isExist                   bool
	)
	for {
		if watcher, err = r.clientset.CoreV1().ReplicationControllers(r.namespace).Watch(r.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if replicationcontrollerList, err = r.List(labelSelector); err != nil {
			return
		}
		if len(replicationcontrollerList.Items) == 0 {
			isExist = false // replicationcontroller not exist
		} else {
			isExist = true // replicationcontroller exist
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
				log.Debug("watch replicationcontroller: bookmark")
			case watch.Error:
				log.Debug("watch replicationcontroller: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch replicationcontroller: reconnect to kubernetes")
		watcher.Stop()
	}
}

// Watch watch replicationcontrollers by name, alias to "WatchByName"
func (r *ReplicationController) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return r.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}

// RunInformer
func (r *ReplicationController) RunInformer(
	addFunc func(obj interface{}),
	updateFunc func(oldObj, newObj interface{}),
	deleteFunc func(obj interface{}),
	stopCh chan struct{}) {
	r.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    addFunc,
		UpdateFunc: updateFunc,
		DeleteFunc: deleteFunc,
	})
	r.informer.Run(stopCh)
}
