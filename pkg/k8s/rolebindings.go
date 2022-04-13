package k8s

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"

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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type RoleBinding struct {
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

// new a RoleBinding handler from kubeconfig or in-cluster config
func NewRoleBinding(ctx context.Context, namespace, kubeconfig string) (rolebinding *RoleBinding, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
	)
	rolebinding = &RoleBinding{}

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

	if len(namespace) == 0 {
		namespace = metav1.NamespaceDefault
	}
	rolebinding.kubeconfig = kubeconfig
	rolebinding.namespace = namespace

	rolebinding.ctx = ctx
	rolebinding.config = config
	rolebinding.restClient = restClient
	rolebinding.clientset = clientset
	rolebinding.dynamicClient = dynamicClient
	rolebinding.discoveryClient = discoveryClient

	rolebinding.Options = &HandlerOptions{}

	return
}
func (r *RoleBinding) Namespace() string {
	return r.namespace
}
func (in *RoleBinding) DeepCopy() *RoleBinding {
	out := new(RoleBinding)

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
func (r *RoleBinding) setNamespace(namespace string) {
	r.Lock()
	defer r.Unlock()
	r.namespace = namespace
}
func (r *RoleBinding) WithNamespace(namespace string) *RoleBinding {
	rolebinding := r.DeepCopy()
	rolebinding.setNamespace(namespace)
	return rolebinding
}
func (r *RoleBinding) WithDryRun() *RoleBinding {
	rb := r.DeepCopy()
	rb.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	rb.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	rb.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	rb.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	rb.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return rb
}
func (r *RoleBinding) SetTimeout(timeout int64) {
	r.Lock()
	defer r.Unlock()
	r.Options.ListOptions.TimeoutSeconds = &timeout
}
func (r *RoleBinding) SetLimit(limit int64) {
	r.Lock()
	defer r.Unlock()
	r.Options.ListOptions.Limit = limit
}
func (r *RoleBinding) SetForceDelete(force bool) {
	r.Lock()
	defer r.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		r.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		r.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create rolebinding from map[string]interface{}
func (r *RoleBinding) CreateFromRaw(raw map[string]interface{}) (*rbacv1.RoleBinding, error) {
	rolebinding := &rbacv1.RoleBinding{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, rolebinding)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(rolebinding.Namespace) != 0 {
		namespace = rolebinding.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.RbacV1().RoleBindings(namespace).Create(r.ctx, rolebinding, r.Options.CreateOptions)
}

// CreateFromBytes create rolebinding from bytes
func (r *RoleBinding) CreateFromBytes(data []byte) (*rbacv1.RoleBinding, error) {

	rolebindingJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	rolebinding := &rbacv1.RoleBinding{}
	err = json.Unmarshal(rolebindingJson, rolebinding)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(rolebinding.Namespace) != 0 {
		namespace = rolebinding.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.RbacV1().RoleBindings(namespace).Create(r.ctx, rolebinding, r.Options.CreateOptions)
}

// CreateFromFile create rolebinding from yaml file
func (r *RoleBinding) CreateFromFile(path string) (*rbacv1.RoleBinding, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.CreateFromBytes(data)
}

// Create create rolebinding from yaml file, alias to "CreateFromFile"
func (r *RoleBinding) Create(path string) (*rbacv1.RoleBinding, error) {
	return r.CreateFromFile(path)
}

// UpdateFromRaw update rolebinding from map[string]interface{}
func (r *RoleBinding) UpdateFromRaw(raw map[string]interface{}) (*rbacv1.RoleBinding, error) {
	rolebinding := &rbacv1.RoleBinding{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, rolebinding)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(rolebinding.Namespace) != 0 {
		namespace = rolebinding.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.RbacV1().RoleBindings(namespace).Update(r.ctx, rolebinding, r.Options.UpdateOptions)
}

// UpdateFromBytes update rolebinding from bytes
func (r *RoleBinding) UpdateFromBytes(data []byte) (*rbacv1.RoleBinding, error) {
	rolebindingJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	rolebinding := &rbacv1.RoleBinding{}
	err = json.Unmarshal(rolebindingJson, rolebinding)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(rolebinding.Namespace) != 0 {
		namespace = rolebinding.Namespace
	} else {
		namespace = r.namespace
	}

	return r.clientset.RbacV1().RoleBindings(namespace).Update(r.ctx, rolebinding, r.Options.UpdateOptions)
}

// UpdateFromFile update rolebinding from yaml file
func (r *RoleBinding) UpdateFromFile(path string) (*rbacv1.RoleBinding, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.UpdateFromBytes(data)
}

// Update update rolebinding from yaml file, alias to "UpdateFromFile"
func (r *RoleBinding) Update(path string) (*rbacv1.RoleBinding, error) {
	return r.UpdateFromFile(path)
}

// ApplyFromRaw apply rolebinding from map[string]interface{}
func (r *RoleBinding) ApplyFromRaw(raw map[string]interface{}) (*rbacv1.RoleBinding, error) {
	rolebinding := &rbacv1.RoleBinding{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, rolebinding)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(rolebinding.Namespace) != 0 {
		namespace = rolebinding.Namespace
	} else {
		namespace = r.namespace
	}

	rolebinding, err = r.clientset.RbacV1().RoleBindings(namespace).Create(r.ctx, rolebinding, r.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		rolebinding, err = r.clientset.RbacV1().RoleBindings(namespace).Update(r.ctx, rolebinding, r.Options.UpdateOptions)
	}
	return rolebinding, err
}

// ApplyFromBytes apply rolebinding from bytes
func (r *RoleBinding) ApplyFromBytes(data []byte) (rolebinding *rbacv1.RoleBinding, err error) {
	rolebinding, err = r.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		rolebinding, err = r.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply rolebinding from yaml file
func (r *RoleBinding) ApplyFromFile(path string) (rolebinding *rbacv1.RoleBinding, err error) {
	rolebinding, err = r.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		rolebinding, err = r.UpdateFromFile(path)
	}
	return
}

// Apply apply rolebinding from yaml file, alias to "ApplyFromFile"
func (r *RoleBinding) Apply(path string) (*rbacv1.RoleBinding, error) {
	return r.ApplyFromFile(path)
}

// DeleteFromBytes delete rolebinding from bytes
func (r *RoleBinding) DeleteFromBytes(data []byte) error {
	rolebindingJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	rolebinding := &rbacv1.RoleBinding{}
	if err = json.Unmarshal(rolebindingJson, rolebinding); err != nil {
		return err
	}

	var namespace string
	if len(rolebinding.Namespace) != 0 {
		namespace = rolebinding.Namespace
	} else {
		namespace = r.namespace
	}

	return r.WithNamespace(namespace).DeleteByName(rolebinding.Name)
}

// DeleteFromFile delete rolebinding from yaml file
func (r *RoleBinding) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return r.DeleteFromBytes(data)
}

// DeleteByName delete rolebinding by name
func (r *RoleBinding) DeleteByName(name string) error {
	return r.clientset.RbacV1().RoleBindings(r.namespace).Delete(r.ctx, name, r.Options.DeleteOptions)
}

// Delete delete rolebinding by name, alias to "DeleteByName"
func (r *RoleBinding) Delete(name string) (err error) {
	return r.DeleteByName(name)
}

// GetFromBytes get rolebinding from bytes
func (r *RoleBinding) GetFromBytes(data []byte) (*rbacv1.RoleBinding, error) {
	rolebindingJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	rolebinding := &rbacv1.RoleBinding{}
	err = json.Unmarshal(rolebindingJson, rolebinding)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(rolebinding.Namespace) != 0 {
		namespace = rolebinding.Namespace
	} else {
		namespace = r.namespace
	}

	return r.WithNamespace(namespace).GetByName(rolebinding.Name)
}

// GetFromFile get rolebinding from yaml file
func (r *RoleBinding) GetFromFile(path string) (*rbacv1.RoleBinding, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.GetFromBytes(data)
}

// GetByName get rolebinding by name
func (r *RoleBinding) GetByName(name string) (*rbacv1.RoleBinding, error) {
	return r.clientset.RbacV1().RoleBindings(r.namespace).Get(r.ctx, name, r.Options.GetOptions)
}
func (r *RoleBinding) Get(name string) (*rbacv1.RoleBinding, error) {
	return r.GetByName(name)
}

// ListByLabel list rolebindings by labels
func (r *RoleBinding) ListByLabel(labels string) (*rbacv1.RoleBindingList, error) {
	listOptions := r.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return r.clientset.RbacV1().RoleBindings(r.namespace).List(r.ctx, *listOptions)
}

// List list rolebindings by labels, alias to  "ListByLabel"
func (r *RoleBinding) List(labels string) (*rbacv1.RoleBindingList, error) {
	return r.ListByLabel(labels)
}

// ListByNamespace list rolebindings by namespace
func (r *RoleBinding) ListByNamespace(namespace string) (*rbacv1.RoleBindingList, error) {
	return r.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all rolebindings in the k8s cluster
func (r *RoleBinding) ListAll() (*rbacv1.RoleBindingList, error) {
	return r.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// WatchByName watch rolebindings by name
func (r *RoleBinding) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: r.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = r.clientset.RbacV1().RoleBindings(r.namespace).Watch(r.ctx, listOptions); err != nil {
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
				log.Debug("watch rolebinding: bookmark.")
			case watch.Error:
				log.Debug("watch rolebinding: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch rolebinding: reconnect to kubernetes")
	}
}

// WatchByLabel watch rolebindings by labelSelector
func (r *RoleBinding) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher         watch.Interface
		rolebindingList *rbacv1.RoleBindingList
		timeout         = int64(0)
		isExist         bool
	)
	for {
		if watcher, err = r.clientset.RbacV1().RoleBindings(r.namespace).Watch(r.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if rolebindingList, err = r.List(labelSelector); err != nil {
			return
		}
		if len(rolebindingList.Items) == 0 {
			isExist = false // rolebinding not exist
		} else {
			isExist = true // rolebinding exist
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
				log.Debug("watch rolebinding: bookmark.")
			case watch.Error:
				log.Debug("watch rolebinding: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch rolebinding: reconnect to kubernetes")
	}
}

// Watch watch rolebindings by name, alias to "WatchByName"
func (r *RoleBinding) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return r.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
