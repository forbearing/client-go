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

type Role struct {
	kubeconfig string
	namespace  string

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

// new a Role handler from kubeconfig or in-cluster config
func NewRole(ctx context.Context, namespace, kubeconfig string) (role *Role, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	role = &Role{}

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

	if len(namespace) == 0 {
		namespace = metav1.NamespaceDefault
	}
	role.kubeconfig = kubeconfig
	role.namespace = namespace
	role.ctx = ctx
	role.config = config
	role.restClient = restClient
	role.clientset = clientset
	role.dynamicClient = dynamicClient
	role.discoveryClient = discoveryClient
	role.informerFactory = informerFactory
	role.Options = &HandlerOptions{}

	return
}
func (r *Role) Namespace() string {
	return r.namespace
}
func (in *Role) DeepCopy() *Role {
	out := new(Role)

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
func (r *Role) setNamespace(namespace string) {
	r.Lock()
	defer r.Unlock()
	r.namespace = namespace
}

func (r *Role) WithNamespace(namespace string) *Role {
	role := r.DeepCopy()
	role.setNamespace(namespace)
	return role
}
func (r *Role) WithDryRun() *Role {
	role := r.DeepCopy()
	role.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	role.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	role.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	role.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	role.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return role
}
func (r *Role) SetTimeout(timeout int64) {
	r.Lock()
	defer r.Unlock()
	r.Options.ListOptions.TimeoutSeconds = &timeout
}
func (r *Role) SetLimit(limit int64) {
	r.Lock()
	defer r.Unlock()
	r.Options.ListOptions.Limit = limit
}
func (r *Role) SetForceDelete(force bool) {
	r.Lock()
	defer r.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		r.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		r.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create role from map[string]interface{}
func (r *Role) CreateFromRaw(raw map[string]interface{}) (*rbacv1.Role, error) {
	role := &rbacv1.Role{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, role)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(role.Namespace) != 0 {
		namespace = role.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.RbacV1().Roles(namespace).Create(r.ctx, role, r.Options.CreateOptions)
}

// CreateFromBytes create role from bytes
func (r *Role) CreateFromBytes(data []byte) (*rbacv1.Role, error) {
	roleJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	role := &rbacv1.Role{}
	err = json.Unmarshal(roleJson, role)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(role.Namespace) != 0 {
		namespace = role.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.RbacV1().Roles(namespace).Create(r.ctx, role, r.Options.CreateOptions)
}

// CreateFromFile create role from yaml file
func (r *Role) CreateFromFile(path string) (*rbacv1.Role, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.CreateFromBytes(data)
}

// Create create role from yaml file, alias to "CreateFromFile"
func (r *Role) Create(path string) (*rbacv1.Role, error) {
	return r.CreateFromFile(path)
}

// UpdateFromRaw update role from map[string]interface{}
func (r *Role) UpdateFromRaw(raw map[string]interface{}) (*rbacv1.Role, error) {
	role := &rbacv1.Role{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, role)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(role.Namespace) != 0 {
		namespace = role.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.RbacV1().Roles(namespace).Update(r.ctx, role, r.Options.UpdateOptions)
}

// UpdateFromBytes update role from bytes
func (r *Role) UpdateFromBytes(data []byte) (*rbacv1.Role, error) {
	roleJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	role := &rbacv1.Role{}
	err = json.Unmarshal(roleJson, role)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(role.Namespace) != 0 {
		namespace = role.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.RbacV1().Roles(namespace).Update(r.ctx, role, r.Options.UpdateOptions)
}

// UpdateFromFile update role from yaml file
func (r *Role) UpdateFromFile(path string) (*rbacv1.Role, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.UpdateFromBytes(data)
}

// Update update role from yaml file, alias to "UpdateFromFile"
func (r *Role) Update(path string) (*rbacv1.Role, error) {
	return r.UpdateFromFile(path)
}

// ApplyFromRaw apply role from map[string]interface{}
func (r *Role) ApplyFromRaw(raw map[string]interface{}) (*rbacv1.Role, error) {
	role := &rbacv1.Role{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, role)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(role.Namespace) != 0 {
		namespace = role.Namespace
	} else {
		namespace = r.namespace
	}

	role, err = r.clientset.RbacV1().Roles(namespace).Create(r.ctx, role, r.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		role, err = r.clientset.RbacV1().Roles(namespace).Update(r.ctx, role, r.Options.UpdateOptions)
	}
	return role, err
}

// ApplyFromBytes apply role from bytes
func (r *Role) ApplyFromBytes(data []byte) (role *rbacv1.Role, err error) {
	role, err = r.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		role, err = r.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply role from yaml file
func (r *Role) ApplyFromFile(path string) (role *rbacv1.Role, err error) {
	role, err = r.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		role, err = r.UpdateFromFile(path)
	}
	return
}

// Apply apply role from yaml file, alias to "ApplyFromFile"
func (r *Role) Apply(path string) (*rbacv1.Role, error) {
	return r.ApplyFromFile(path)
}

// DeleteFromBytes delete role from bytes
func (r *Role) DeleteFromBytes(data []byte) error {
	roleJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	role := &rbacv1.Role{}
	err = json.Unmarshal(roleJson, role)
	if err != nil {
		return err
	}

	var namespace string
	if len(role.Namespace) != 0 {
		namespace = role.Namespace
	} else {
		namespace = r.namespace
	}

	return r.WithNamespace(namespace).DeleteByName(role.Name)
}

// DeleteFromFile delete role from yaml file
func (r *Role) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return r.DeleteFromBytes(data)
}

// DeleteByName delete role by name
func (r *Role) DeleteByName(name string) error {
	return r.clientset.RbacV1().Roles(r.namespace).Delete(r.ctx, name, r.Options.DeleteOptions)
}

// Delete delete role by name, alias to "DeleteByName"
func (r *Role) Delete(name string) error {
	return r.DeleteByName(name)
}

// GetFromBytes get role from bytes
func (r *Role) GetFromBytes(data []byte) (*rbacv1.Role, error) {
	roleJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	role := &rbacv1.Role{}
	err = json.Unmarshal(roleJson, role)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(role.Namespace) != 0 {
		namespace = role.Namespace
	} else {
		namespace = r.namespace
	}

	return r.WithNamespace(namespace).GetByName(role.Name)
}

// GetFromFile get role from yamlfile
func (r *Role) GetFromFile(path string) (*rbacv1.Role, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.GetFromBytes(data)
}

// GetByName get role by name
func (r *Role) GetByName(name string) (*rbacv1.Role, error) {
	return r.clientset.RbacV1().Roles(r.namespace).Get(r.ctx, name, r.Options.GetOptions)
}

// Get get role by name, alias to "GetByName"
func (r *Role) Get(name string) (*rbacv1.Role, error) {
	return r.GetByName(name)
}

// ListByLabel list roles by labels
func (r *Role) ListByLabel(labels string) (*rbacv1.RoleList, error) {
	listOptions := r.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return r.clientset.RbacV1().Roles(r.namespace).List(r.ctx, *listOptions)
}

// List list roles by labels, alias to "ListByLabel"
func (r *Role) List(labels string) (*rbacv1.RoleList, error) {
	return r.ListByLabel(labels)
}

// ListByNamespace list roles by namespace
func (r *Role) ListByNamespace(namespace string) (*rbacv1.RoleList, error) {
	return r.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all roles in the k8s cluster
func (r *Role) ListAll() (*rbacv1.RoleList, error) {
	return r.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// WatchByName watch roles by name
func (r *Role) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: r.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = r.clientset.RbacV1().Roles(r.namespace).Watch(r.ctx, listOptions); err != nil {
			return
		}
		if _, err = r.Get(name); err != nil {
			isExist = false // role not exist
		} else {
			isExist = true // role exist
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
				log.Debug("watch role: bookmark.")
			case watch.Error:
				log.Debug("watch role: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch role: reconnect to kubernetes")
	}
}

// WatchByLabel watch roles by labelSelector
func (r *Role) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher  watch.Interface
		roleList *rbacv1.RoleList
		timeout  = int64(0)
		isExist  bool
	)
	for {
		if watcher, err = r.clientset.RbacV1().Roles(r.namespace).Watch(r.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if roleList, err = r.List(labelSelector); err != nil {
			return
		}
		if len(roleList.Items) == 0 {
			isExist = false // role not exist
		} else {
			isExist = true // role exist
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
				log.Debug("watch role: bookmark.")
			case watch.Error:
				log.Debug("watch role: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch role: reconnect to kubernetes")
	}
}

// Watch watch roles by name, alias to "WatchByName"
func (r *Role) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return r.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
