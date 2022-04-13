package k8s

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"

	log "github.com/sirupsen/logrus"
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

type ConfigMap struct {
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

// new a configmap handler from kubeconfig or in-cluster config
func NewConfigMap(ctx context.Context, namespace, kubeconfig string) (configmap *ConfigMap, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
	)
	configmap = &ConfigMap{}

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

	if len(namespace) == 0 {
		namespace = metav1.NamespaceDefault
	}
	configmap.kubeconfig = kubeconfig
	configmap.namespace = namespace

	configmap.ctx = ctx
	configmap.config = config
	configmap.restClient = restClient
	configmap.clientset = clientset
	configmap.dynamicClient = dynamicClient
	configmap.discoveryClient = discoveryClient

	configmap.Options = &HandlerOptions{}

	return
}
func (c *ConfigMap) Namespace() string {
	return c.namespace
}
func (in *ConfigMap) DeepCopy() *ConfigMap {
	out := new(ConfigMap)

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
func (c *ConfigMap) setNamespace(namespace string) {
	c.Lock()
	defer c.Unlock()
	c.namespace = namespace
}
func (c *ConfigMap) WithNamespace(namespace string) *ConfigMap {
	cm := c.DeepCopy()
	cm.setNamespace(namespace)
	return cm
}
func (c *ConfigMap) WithDryRun() *ConfigMap {
	configmap := c.DeepCopy()
	configmap.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	configmap.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	configmap.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	configmap.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	configmap.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return configmap
}
func (c *ConfigMap) SetTimeout(timeout int64) {
	c.Lock()
	defer c.Unlock()
	c.Options.ListOptions.TimeoutSeconds = &timeout
}
func (c *ConfigMap) SetLimit(limit int64) {
	c.Lock()
	defer c.Unlock()
	c.Options.ListOptions.Limit = limit
}

func (c *ConfigMap) SetForceDelete(force bool) {
	c.Lock()
	defer c.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		c.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		c.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// create configmap from bytes
func (c *ConfigMap) CreateFromBytes(data []byte) (*corev1.ConfigMap, error) {
	cmJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	configmap := &corev1.ConfigMap{}
	err = json.Unmarshal(cmJson, configmap)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(configmap.Namespace) != 0 {
		namespace = configmap.Namespace
	} else {
		namespace = c.namespace
	}

	return c.clientset.CoreV1().ConfigMaps(namespace).Create(c.ctx, configmap, c.Options.CreateOptions)
}

// create configmap from file
func (c *ConfigMap) CreateFromFile(path string) (*corev1.ConfigMap, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.CreateFromBytes(data)
}

// create configmap from file, alias to "CreateFromFile"
func (c *ConfigMap) Create(path string) (*corev1.ConfigMap, error) {
	return c.CreateFromFile(path)
}

// update configmap from bytes
func (c *ConfigMap) UpdateFromBytes(data []byte) (*corev1.ConfigMap, error) {
	cmJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	configmap := &corev1.ConfigMap{}
	err = json.Unmarshal(cmJson, configmap)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(configmap.Namespace) != 0 {
		namespace = configmap.Namespace
	} else {
		namespace = c.namespace
	}

	return c.clientset.CoreV1().ConfigMaps(namespace).Update(c.ctx, configmap, c.Options.UpdateOptions)
}

// update configmap from file
func (c *ConfigMap) UpdateFromFile(path string) (*corev1.ConfigMap, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.UpdateFromBytes(data)
}

// update configmap from file, alias to "UpdateFromFile"
func (c *ConfigMap) Update(path string) (*corev1.ConfigMap, error) {
	return c.UpdateFromFile(path)
}

// apply configmap from bytes
func (c *ConfigMap) ApplyFromBytes(data []byte) (configmap *corev1.ConfigMap, err error) {
	configmap, err = c.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		configmap, err = c.UpdateFromBytes(data)
	}
	return
}

// apply configmap from file
func (c *ConfigMap) ApplyFromFile(path string) (configmap *corev1.ConfigMap, err error) {
	configmap, err = c.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		configmap, err = c.UpdateFromFile(path)
	}
	return
}

// apply configmap from file, alias to "ApplyFromFile"
func (c *ConfigMap) Apply(path string) (*corev1.ConfigMap, error) {
	return c.ApplyFromFile(path)
}

// delete configmap from bytes
func (c *ConfigMap) DeleteFromBytes(data []byte) error {
	cmJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	configmap := &corev1.ConfigMap{}
	err = json.Unmarshal(cmJson, configmap)
	if err != nil {
		return err
	}

	var namespace string
	if len(configmap.Namespace) != 0 {
		namespace = configmap.Namespace
	} else {
		namespace = c.namespace
	}

	return c.WithNamespace(namespace).DeleteByName(configmap.Name)
}

// delete configmap from file
func (c *ConfigMap) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return c.DeleteFromBytes(data)
}

// delete configmap by name
func (c *ConfigMap) DeleteByName(name string) error {
	return c.clientset.CoreV1().ConfigMaps(c.namespace).Delete(c.ctx, name, c.Options.DeleteOptions)
}

// delete configmap by name, alias to "DeleteByName"
func (c *ConfigMap) Delete(name string) error {
	return c.DeleteByName(name)
}

// get configmap from bytes
func (c *ConfigMap) GetFromBytes(data []byte) (*corev1.ConfigMap, error) {
	cmJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	configmap := &corev1.ConfigMap{}
	err = json.Unmarshal(cmJson, configmap)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(configmap.Namespace) != 0 {
		namespace = configmap.Namespace
	} else {
		namespace = c.namespace
	}

	return c.WithNamespace(namespace).GetByName(configmap.Name)
}

// get configmap from file
func (c *ConfigMap) GetFromFile(path string) (configmap *corev1.ConfigMap, err error) {
	var data []byte
	if data, err = ioutil.ReadFile(path); err != nil {
		return
	}
	configmap, err = c.GetFromBytes(data)
	return
}

// get configmap by name
func (c *ConfigMap) GetByName(name string) (*corev1.ConfigMap, error) {
	return c.clientset.CoreV1().ConfigMaps(c.namespace).Get(c.ctx, name, c.Options.GetOptions)
}

// get configmap by name
func (c *ConfigMap) Get(name string) (*corev1.ConfigMap, error) {
	return c.GetByName(name)
}

// list configmaps by labels
func (c *ConfigMap) ListByLabel(labels string) (*corev1.ConfigMapList, error) {
	listOptions := c.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return c.clientset.CoreV1().ConfigMaps(c.namespace).List(c.ctx, *listOptions)
}

// list configmaps by labels, alias to "ListByLabel"
func (c *ConfigMap) List(labels string) (*corev1.ConfigMapList, error) {
	return c.ListByLabel(labels)
}

// list configmaps by namespace
func (c *ConfigMap) ListByNamespace(namespace string) (*corev1.ConfigMapList, error) {
	return c.WithNamespace(namespace).ListByLabel("")
}

// list all configmaps in the k8s cluster
func (c *ConfigMap) ListAll() (*corev1.ConfigMapList, error) {
	return c.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// get configmap .spec.data
func (c *ConfigMap) GetData(name string) (map[string]string, error) {
	data := make(map[string]string)
	configmap, err := c.Get(name)
	if err != nil {
		return data, err
	}
	data = configmap.Data
	return data, nil
}

// watch configmap by name
func (c *ConfigMap) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: c.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = c.clientset.CoreV1().ConfigMaps(c.namespace).Watch(c.ctx, listOptions); err != nil {
			return
		}
		if _, err = c.Get(name); err != nil {
			isExist = false // configmap not exist
		} else {
			isExist = true // configmap exist
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
				log.Debug("watch configmap: bookmark")
			case watch.Error:
				log.Debug("watch configmap: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch configmap: reconnect to kubernetes")
		watcher.Stop()
	}
}

// watch configmap by labelSelector
func (c *ConfigMap) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher       watch.Interface
		configmapList *corev1.ConfigMapList
		timeout       = int64(0)
		isExist       bool
	)
	for {
		if watcher, err = c.clientset.CoreV1().ConfigMaps(c.namespace).Watch(c.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if configmapList, err = c.List(labelSelector); err != nil {
			return
		}
		if len(configmapList.Items) == 0 {
			isExist = false // configmap not exist
		} else {
			isExist = true // configmap exist
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
				log.Debug("watch configmap: bookmark")
			case watch.Error:
				log.Debug("watch configmap: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch configmap: reconnect to kubernetes")
		watcher.Stop()
	}
}

// watch configmap by name, alias to "WatchByName"
func (c *ConfigMap) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return c.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
