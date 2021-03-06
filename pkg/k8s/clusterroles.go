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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterRole struct {
	kubeconfig string

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

// new a clusterrole handler from kubeconfig or in-cluster config
func NewClusterRole(ctx context.Context, kubeconfig string) (clusterrole *ClusterRole, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	clusterrole = &ClusterRole{}

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
	config.GroupVersion = &rbacv1.SchemeGroupVersion
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

	clusterrole.kubeconfig = kubeconfig
	clusterrole.ctx = ctx
	clusterrole.config = config
	clusterrole.restClient = restClient
	clusterrole.clientset = clientset
	clusterrole.dynamicClient = dynamicClient
	clusterrole.discoveryClient = discoveryClient
	clusterrole.informerFactory = informerFactory
	clusterrole.informer = informerFactory.Rbac().V1().ClusterRoles().Informer()
	clusterrole.Options = &HandlerOptions{}

	return
}
func (in *ClusterRole) DeepCopy() *ClusterRole {
	out := new(ClusterRole)

	out.kubeconfig = in.kubeconfig

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
func (c *ClusterRole) WithDryRun() *ClusterRole {
	cr := c.DeepCopy()
	cr.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	cr.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	cr.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	cr.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	cr.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return cr
}
func (c *ClusterRole) SetTimeout(timeout int64) {
	c.Lock()
	defer c.Unlock()
	c.Options.ListOptions.TimeoutSeconds = &timeout
}
func (c *ClusterRole) SetLimit(limit int64) {
	c.Lock()
	defer c.Unlock()
	c.Options.ListOptions.Limit = limit
}
func (c *ClusterRole) SetForceDelete(force bool) {
	c.Lock()
	defer c.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		c.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		c.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create clusterrole from map[string]interface{}
func (c *ClusterRole) CreateFromRaw(raw map[string]interface{}) (*rbacv1.ClusterRole, error) {
	cr := &rbacv1.ClusterRole{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, cr)
	if err != nil {
		return nil, err
	}

	return c.clientset.RbacV1().ClusterRoles().Create(c.ctx, cr, c.Options.CreateOptions)
}

// CreateFromBytes create clusterrole from bytes
func (c *ClusterRole) CreateFromBytes(data []byte) (*rbacv1.ClusterRole, error) {
	crJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	cr := &rbacv1.ClusterRole{}
	if err = json.Unmarshal(crJson, cr); err != nil {
		return nil, err
	}

	return c.clientset.RbacV1().ClusterRoles().Create(c.ctx, cr, c.Options.CreateOptions)
}

// CreateFromFile create clusterrole from yaml file
func (c *ClusterRole) CreateFromFile(path string) (*rbacv1.ClusterRole, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.CreateFromBytes(data)
}

// Create create clusterrole from yaml file, alias to "CreateFromFile"
func (c *ClusterRole) Create(path string) (*rbacv1.ClusterRole, error) {
	return c.CreateFromFile(path)
}

// UpdateFromRaw update clusterrole from map[string]interface{}
func (c *ClusterRole) UpdateFromRaw(raw map[string]interface{}) (*rbacv1.ClusterRole, error) {
	cr := &rbacv1.ClusterRole{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, cr)
	if err != nil {
		return nil, err
	}

	return c.clientset.RbacV1().ClusterRoles().Update(c.ctx, cr, c.Options.UpdateOptions)
}

// UpdateFromBytes update clusterrole from bytes
func (c *ClusterRole) UpdateFromBytes(data []byte) (*rbacv1.ClusterRole, error) {
	crJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	cr := &rbacv1.ClusterRole{}
	if err = json.Unmarshal(crJson, cr); err != nil {
		return nil, err
	}

	return c.clientset.RbacV1().ClusterRoles().Update(c.ctx, cr, c.Options.UpdateOptions)
}

// UpdateFromFile update clusterrole from yaml file
func (c *ClusterRole) UpdateFromFile(path string) (*rbacv1.ClusterRole, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.UpdateFromBytes(data)
}

// Update update clusterrole from yaml file, alias to "UpdateFromFile"
func (c *ClusterRole) Update(path string) (*rbacv1.ClusterRole, error) {
	return c.UpdateFromFile(path)
}

// ApplyFromRaw apply clusterrole from map[string]interface{}
func (p *ClusterRole) applyFromRaw(raw map[string]interface{}) (*rbacv1.ClusterRole, error) {
	cr := &rbacv1.ClusterRole{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, cr)
	if err != nil {
		return nil, err
	}

	cr, err = p.clientset.RbacV1().ClusterRoles().Create(p.ctx, cr, p.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		cr, err = p.clientset.RbacV1().ClusterRoles().Update(p.ctx, cr, p.Options.UpdateOptions)
	}
	return cr, err
}

// ApplyFromBytes apply clusterrole from bytes
func (c *ClusterRole) ApplyFromBytes(data []byte) (clusterrole *rbacv1.ClusterRole, err error) {
	clusterrole, err = c.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		clusterrole, err = c.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply clusterrole from yaml file
func (c *ClusterRole) ApplyFromFile(path string) (clusterrole *rbacv1.ClusterRole, err error) {
	clusterrole, err = c.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		clusterrole, err = c.UpdateFromFile(path)
	}
	return
}

// Apply apply clusterrole from yaml file, alias to "ApplyFromFile"
func (c *ClusterRole) Apply(path string) (*rbacv1.ClusterRole, error) {
	return c.ApplyFromFile(path)
}

// DeleteFromBytes delete clusterrole from bytes
func (c *ClusterRole) DeleteFromBytes(data []byte) error {
	crJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	cr := &rbacv1.ClusterRole{}
	err = json.Unmarshal(crJson, cr)
	if err != nil {
		return err
	}

	return c.DeleteByName(cr.Name)
}

// DeleteFromFile delete clusterrole from yaml file
func (c *ClusterRole) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return c.DeleteFromBytes(data)
}

// DeleteByName delete clusterrole by name
func (c *ClusterRole) DeleteByName(name string) (err error) {
	return c.clientset.RbacV1().ClusterRoles().Delete(c.ctx, name, c.Options.DeleteOptions)
}

// Delete delete clusterrole by name, alias to "DeleteByName"
func (c *ClusterRole) Delete(name string) (err error) {
	return c.DeleteByName(name)
}

// Get get clusterrole from bytes
func (c *ClusterRole) GetFromBytes(data []byte) (*rbacv1.ClusterRole, error) {
	crJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	cr := &rbacv1.ClusterRole{}
	err = json.Unmarshal(crJson, cr)
	if err != nil {
		return nil, err
	}

	return c.GetByName(cr.Name)
}

// GetFromFile get clusterrole from yaml file
func (c *ClusterRole) GetFromFile(path string) (*rbacv1.ClusterRole, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.GetFromBytes(data)
}

// Get get clusterrole by name
func (c *ClusterRole) GetByName(name string) (*rbacv1.ClusterRole, error) {
	return c.clientset.RbacV1().ClusterRoles().Get(c.ctx, name, c.Options.GetOptions)
}

// Get get clusterrole by name, alias to "GetByName"
func (c *ClusterRole) Get(name string) (*rbacv1.ClusterRole, error) {
	return c.GetByName(name)
}

// ListByLabel list clusterroles by labels
func (c *ClusterRole) ListByLabel(labels string) (*rbacv1.ClusterRoleList, error) {
	listOptions := c.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return c.clientset.RbacV1().ClusterRoles().List(c.ctx, *listOptions)
}

// List list clusterroles by labels, alias to "ListByLabel"
func (c *ClusterRole) List(labels string) (*rbacv1.ClusterRoleList, error) {
	return c.ListByLabel(labels)
}

// ListAll list all clusterroles in the k8s cluster
func (c *ClusterRole) ListAll(labels string) (*rbacv1.ClusterRoleList, error) {
	return c.ListByLabel("")
}

// WatchByName watch clusterroles by name
func (c *ClusterRole) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = c.clientset.RbacV1().ClusterRoles().Watch(c.ctx, listOptions); err != nil {
			return
		}
		if _, err = c.Get(name); err != nil {
			isExist = false // clusterroles not exist
		} else {
			isExist = true // clusterroles exist
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
				log.Debug("watch clusterrole: bookmark.")
			case watch.Error:
				log.Debug("watch clusterrole: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch clusterrole: reconnect to kubernetes")
	}
}

// WatchByLabel watch clusterroles by labelSelector
func (c *ClusterRole) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher         watch.Interface
		clusterroleList *rbacv1.ClusterRoleList
		timeout         = int64(0)
		isExist         bool
	)
	for {
		if watcher, err = c.clientset.RbacV1().ClusterRoles().Watch(c.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if clusterroleList, err = c.List(labelSelector); err != nil {
			return
		}
		if len(clusterroleList.Items) == 0 {
			isExist = false // clusterrole not exist
		} else {
			isExist = true // clusterrole exist
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
				log.Debug("watch clusterrole: bookmark.")
			case watch.Error:
				log.Debug("watch clusterrole: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch clusterrole: reconnect to kubernetes")
	}
}

// Watch watch clusterroles by name, alias to "WatchByName"
func (c *ClusterRole) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return c.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}

// RunInformer
func (c *ClusterRole) RunInformer(
	addFunc func(obj interface{}),
	updateFunc func(oldObj, newObj interface{}),
	deleteFunc func(obj interface{}),
	stopCh chan struct{}) {
	c.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    addFunc,
		UpdateFunc: updateFunc,
		DeleteFunc: deleteFunc,
	})
	c.informer.Run(stopCh)
}
