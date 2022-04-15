package k8s

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
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
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterRoleBinding struct {
	kubeconfig string

	ctx             context.Context
	config          *rest.Config
	restClient      *rest.RESTClient
	clientset       *kubernetes.Clientset
	dynamicClient   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient
	informerFactory informers.SharedInformerFactory

	Options *HandlerOptions

	sync.Mutex
}

// new a clusterrolebinding handler from kubeconfig or in-cluster config
func NewClusterRoleBinding(ctx context.Context, kubeconfig string) (crb *ClusterRoleBinding, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	crb = &ClusterRoleBinding{}

	// create rest config
	if len(kubeconfig) != 0 {
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	} else {
		// create the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	// setup APIPath, GroupVersion and NegotiatedSerializer before initializing a RESTClient
	config.APIPath = "api"
	config.GroupVersion = &rbacv1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs
	// create a RESTClient for the given config
	restClient, err = rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}
	// create a Clientset for the given config
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	// create a dynamic client for the given config
	dynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	// create a DiscoveryClient for the given config
	discoveryClient, err = discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	// create a sharedInformerFactory for all namespaces.
	informerFactory = informers.NewSharedInformerFactory(clientset, time.Minute)

	crb.kubeconfig = kubeconfig
	crb.ctx = ctx
	crb.config = config
	crb.restClient = restClient
	crb.clientset = clientset
	crb.dynamicClient = dynamicClient
	crb.discoveryClient = discoveryClient
	crb.informerFactory = informerFactory
	crb.Options = &HandlerOptions{}

	return crb, nil
}
func (in *ClusterRoleBinding) DeepCopy() *ClusterRoleBinding {
	out := new(ClusterRoleBinding)

	out.kubeconfig = in.kubeconfig

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
func (c *ClusterRoleBinding) WithDryRun() *ClusterRoleBinding {
	crb := c.DeepCopy()
	crb.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	crb.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	crb.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	crb.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	crb.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return crb
}
func (c *ClusterRoleBinding) SetTimeout(timeout int64) {
	c.Lock()
	defer c.Unlock()
	c.Options.ListOptions.TimeoutSeconds = &timeout
}
func (c *ClusterRoleBinding) SetLimit(limit int64) {
	c.Lock()
	defer c.Unlock()
	c.Options.ListOptions.Limit = limit
}
func (c *ClusterRoleBinding) SetForceDelete(force bool) {
	c.Lock()
	defer c.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		c.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		c.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create ClusterRoleBinding from map[string]interface{}
func (c *ClusterRoleBinding) CreateFromRaw(raw map[string]interface{}) (*rbacv1.ClusterRoleBinding, error) {
	crb := &rbacv1.ClusterRoleBinding{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, crb)
	if err != nil {
		return nil, err
	}

	return c.clientset.RbacV1().ClusterRoleBindings().Create(c.ctx, crb, c.Options.CreateOptions)
}

// CreateFromBytes create clusterrolebinding from bytes
func (c *ClusterRoleBinding) CreateFromBytes(data []byte) (*rbacv1.ClusterRoleBinding, error) {
	crbJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	crb := &rbacv1.ClusterRoleBinding{}
	err = json.Unmarshal(crbJson, crb)
	if err != nil {
		return nil, err
	}

	return c.clientset.RbacV1().ClusterRoleBindings().Create(c.ctx, crb, c.Options.CreateOptions)
}

// CreateFromFile create clusterrolebinding from yaml file
func (c *ClusterRoleBinding) CreateFromFile(path string) (*rbacv1.ClusterRoleBinding, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.CreateFromBytes(data)
}

// Create create clusterrolebinding from yaml file, alias to "CreateFromFile"
func (c *ClusterRoleBinding) Create(path string) (*rbacv1.ClusterRoleBinding, error) {
	return c.CreateFromFile(path)
}

// UpdateFromRaw update clusterrolebinding from map[string]interface{}
func (c *ClusterRoleBinding) UpdateFromRaw(raw map[string]interface{}) (*rbacv1.ClusterRoleBinding, error) {
	crb := &rbacv1.ClusterRoleBinding{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, crb)
	if err != nil {
		return nil, err
	}

	return c.clientset.RbacV1().ClusterRoleBindings().Update(c.ctx, crb, c.Options.UpdateOptions)
}

// UpdateFromBytes update clusterrolebinding from bytes
func (c *ClusterRoleBinding) UpdateFromBytes(data []byte) (*rbacv1.ClusterRoleBinding, error) {
	crbJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	crb := &rbacv1.ClusterRoleBinding{}
	err = json.Unmarshal(crbJson, crb)
	if err != nil {
		return nil, err
	}

	return c.clientset.RbacV1().ClusterRoleBindings().Update(c.ctx, crb, c.Options.UpdateOptions)
}

// UpdateFromFile update clusterrolebinding from yaml file
func (c *ClusterRoleBinding) UpdateFromFile(path string) (*rbacv1.ClusterRoleBinding, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.UpdateFromBytes(data)
}

// Update update clusterrolebinding from file, alias to "UpdateFromFile"
func (c *ClusterRoleBinding) Update(path string) (*rbacv1.ClusterRoleBinding, error) {
	return c.UpdateFromFile(path)
}

// ApplyFromRaw apply clusterrolebinding from map[string]interface{}
func (c *ClusterRoleBinding) ApplyFromRaw(raw map[string]interface{}) (*rbacv1.ClusterRoleBinding, error) {
	crb := &rbacv1.ClusterRoleBinding{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, crb)
	if err != nil {
		return nil, err
	}

	crb, err = c.clientset.RbacV1().ClusterRoleBindings().Create(c.ctx, crb, c.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		crb, err = c.clientset.RbacV1().ClusterRoleBindings().Update(c.ctx, crb, c.Options.UpdateOptions)
	}
	return crb, err
}

// ApplyFromBytes apply clusterrolebinding from bytes
func (c *ClusterRoleBinding) ApplyFromBytes(data []byte) (crb *rbacv1.ClusterRoleBinding, err error) {
	crb, err = c.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		crb, err = c.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply clusterrolebinding from yaml file
func (c *ClusterRoleBinding) ApplyFromFile(path string) (crb *rbacv1.ClusterRoleBinding, err error) {
	crb, err = c.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		crb, err = c.UpdateFromFile(path)
	}
	return
}

// Apply apply clusterrolebinding from file, alias to "ApplyFromFile"
func (c *ClusterRoleBinding) Apply(path string) (*rbacv1.ClusterRoleBinding, error) {
	return c.ApplyFromFile(path)
}

// DeleteFromBytes delete clusterrolebinding from bytes
func (c *ClusterRoleBinding) DeleteFromBytes(data []byte) error {
	crbJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	crb := &rbacv1.ClusterRoleBinding{}
	err = json.Unmarshal(crbJson, crb)
	if err != nil {
		return err
	}

	return c.DeleteByName(crb.Name)
}

// DeleteFromFile delete clusterrolebinding from yaml file
func (c *ClusterRoleBinding) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return c.DeleteFromBytes(data)
}

// DeleteByName delete clusterrolebinding by name
func (c *ClusterRoleBinding) DeleteByName(name string) error {
	return c.clientset.RbacV1().ClusterRoleBindings().Delete(c.ctx, name, c.Options.DeleteOptions)
}

// Delete delete clusterrolebinding by name, alias to "DeleteByName"
func (c *ClusterRoleBinding) Delete(name string) error {
	return c.DeleteByName(name)
}

// GetFromBytes get clusterrolebinding from bytes
func (c *ClusterRoleBinding) GetFromBytes(data []byte) (*rbacv1.ClusterRoleBinding, error) {
	crbJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	crb := &rbacv1.ClusterRoleBinding{}
	err = json.Unmarshal(crbJson, crb)
	if err != nil {
		return nil, err
	}

	return c.GetByName(crb.Name)
}

// GetFromFile get clusterrolebinding from yaml file
func (c *ClusterRoleBinding) GetFromFile(path string) (*rbacv1.ClusterRoleBinding, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.GetFromBytes(data)
}

// GetByName get clusterrolebinding by name
func (c *ClusterRoleBinding) GetByName(name string) (*rbacv1.ClusterRoleBinding, error) {
	return c.clientset.RbacV1().ClusterRoleBindings().Get(c.ctx, name, c.Options.GetOptions)
}

// Get get clusterrolebinding by name, alias to "GetByName"
func (c *ClusterRoleBinding) Get(name string) (*rbacv1.ClusterRoleBinding, error) {
	return c.clientset.RbacV1().ClusterRoleBindings().Get(c.ctx, name, c.Options.GetOptions)
}

// ListByLabel list clusterrolebindings by labels
func (c *ClusterRoleBinding) ListByLabel(labels string) (*rbacv1.ClusterRoleBindingList, error) {
	listOptions := c.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return c.clientset.RbacV1().ClusterRoleBindings().List(c.ctx, *listOptions)
}

// List list clusterrolebindings by labels, alias to "ListByLabel"
func (c *ClusterRoleBinding) List(labels string) (*rbacv1.ClusterRoleBindingList, error) {
	return c.ListByLabel(labels)
}

// ListAll list all clusterrolebindings in the k8s cluster
func (c *ClusterRoleBinding) ListAll() (*rbacv1.ClusterRoleBindingList, error) {
	return c.ListByLabel("")
}

// WatchByName  watch clusterrolebindings by name
func (c *ClusterRoleBinding) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = c.clientset.RbacV1().ClusterRoleBindings().Watch(c.ctx, listOptions); err != nil {
			return
		}
		if _, err = c.Get(name); err != nil {
			isExist = false // clusterrolebinding not exist
		} else {
			isExist = true // clusterrolebinding exist
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
				log.Debug("watch clusterrolebinding: bookmark.")
			case watch.Error:
				log.Debug("watch clusterrolebinding: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch clusterrolebinding: reconnect to kubernetes")
	}
}

// WatchByLabel watch clusterrolebindings by labelSelector
func (c *ClusterRoleBinding) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher                watch.Interface
		clusterrolebindingList *rbacv1.ClusterRoleBindingList
		timeout                = int64(0)
		isExist                bool
	)
	for {
		if watcher, err = c.clientset.RbacV1().ClusterRoleBindings().Watch(c.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if clusterrolebindingList, err = c.List(labelSelector); err != nil {
			return
		}
		if len(clusterrolebindingList.Items) == 0 {
			isExist = false // clusterrolebinding not exist
		} else {
			isExist = true // clusterrolebinding exist
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
				log.Debug("watch clusterrolebinding: bookmark.")
			case watch.Error:
				log.Debug("watch clusterrolebinding: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch clusterrolebinding: reconnect to kubernetes")
	}
}

// Watch watch clusterrolebinding by name, alias to "WatchByName"
func (c *ClusterRoleBinding) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) error {
	return c.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
