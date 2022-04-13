package k8s

// TODO:
// 1. GetPods 目前是通过 matchLabels, 还需要考虑 matchExpressions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	serializeryaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/util/yaml"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"

	//_ "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	//_ "k8s.io/client-go/applyconfigurations/apps/v1"
	//_ "k8s.io/client-go/applyconfigurations/meta/v1"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type Deployment struct {
	kubeconfig string
	namespace  string

	ctx                context.Context
	config             *rest.Config
	restClient         *rest.RESTClient
	clientset          *kubernetes.Clientset
	dynamicClient      dynamic.Interface
	discoveryClient    *discovery.DiscoveryClient
	discoveryInterface discovery.DiscoveryInterface

	Options *HandlerOptions

	sync.Mutex
}

//// Discovery retrieves the DiscoveryClient
//func (c *Clientset) Discovery() discovery.DiscoveryInterface {
//    if c == nil {
//        return nil
//    }
//    return c.DiscoveryClient
//}
// clientset 调用 Discovery 方法可以得到一个 discovery.DiscoveryInterface
// discovery.DiscoveryClient 其实就是 discovery.DiscoveryInterface 的一个实现
// new a deployment handler from kubeconfig or in-cluster config
func NewDeployment(ctx context.Context, namespace, kubeconfig string) (deployment *Deployment, err error) {
	var (
		config             *rest.Config
		restClient         *rest.RESTClient
		clientset          *kubernetes.Clientset
		dynamicClient      dynamic.Interface
		discoveryClient    *discovery.DiscoveryClient
		discoveryInterface discovery.DiscoveryInterface
	)
	deployment = &Deployment{}

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
	config.GroupVersion = &appsv1.SchemeGroupVersion
	//config.GroupVersion = &schema.GroupVersion{Group: "apps", Version: "v1"}
	config.NegotiatedSerializer = scheme.Codecs
	//config.UserAgent = rest.DefaultKubernetesUserAgent()
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
	//discoveryClient = clientset.DiscoveryClient
	//discoveryInterface = clientset.Discovery()

	// default namespace is meatv1.NamespaceDefault ("default")
	if len(namespace) == 0 {
		namespace = metav1.NamespaceDefault
	}
	deployment.kubeconfig = kubeconfig
	deployment.namespace = namespace

	deployment.ctx = ctx
	deployment.config = config
	deployment.restClient = restClient
	deployment.clientset = clientset
	deployment.dynamicClient = dynamicClient
	deployment.discoveryClient = discoveryClient
	//deployment.discoveryInterface = discoveryInterface
	_ = discoveryInterface

	deployment.Options = &HandlerOptions{}

	return deployment, nil
}
func (d *Deployment) Namespace() string {
	return d.namespace
}
func (in *Deployment) DeepCopy() *Deployment {
	out := new(Deployment)

	// 值拷贝即是深拷贝
	out.kubeconfig = in.kubeconfig
	out.namespace = in.namespace

	// 和几个字段都是共用的, 不需要深拷贝
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

	// 锁 sync.Mutex 不需要拷贝, 也不能拷贝. 拷贝 sync.Mutex 会直接 panic

	//fmt.Printf("%#v\n", oldHandler)
	//fmt.Println()
	//fmt.Printf("%#v\n", out)
	//select {}

	return out
}
func (d *Deployment) setNamespace(namespace string) {
	d.Lock()
	d.Unlock()
	d.namespace = namespace
}
func (d *Deployment) WithNamespace(namespace string) *Deployment {
	deploy := d.DeepCopy()
	deploy.setNamespace(namespace)
	return deploy
}
func (d *Deployment) WithDryRun() *Deployment {
	deploy := d.DeepCopy()
	deploy.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	deploy.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	deploy.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	deploy.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	deploy.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return deploy
}
func (d *Deployment) SetTimeout(timeout int64) {
	d.Lock()
	defer d.Unlock()
	d.Options.ListOptions.TimeoutSeconds = &timeout
}
func (d *Deployment) SetLimit(limit int64) {
	d.Lock()
	defer d.Unlock()
	d.Options.ListOptions.Limit = limit
}
func (d *Deployment) SetForceDelete(force bool) {
	d.Lock()
	defer d.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		d.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		d.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create deployment from map[string]interface{}
func (d *Deployment) CreateFromRaw(raw map[string]interface{}) (*appsv1.Deployment, error) {
	deploy := &appsv1.Deployment{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, deploy)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(deploy.Namespace) != 0 {
		namespace = deploy.Namespace
	} else {
		namespace = d.namespace
	}

	return d.clientset.AppsV1().Deployments(namespace).Create(d.ctx, deploy, d.Options.CreateOptions)
}

// CreateFromBytes create deployment from bytes
func (d *Deployment) CreateFromBytes(data []byte) (*appsv1.Deployment, error) {
	deployJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	deploy := &appsv1.Deployment{}
	err = json.Unmarshal(deployJson, deploy)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(deploy.Namespace) != 0 {
		namespace = deploy.Namespace
	} else {
		namespace = d.namespace
	}

	return d.clientset.AppsV1().Deployments(namespace).Create(d.ctx, deploy, d.Options.CreateOptions)
}

// CreateFromFile create deployment from yaml file
func (d *Deployment) CreateFromFile(path string) (*appsv1.Deployment, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return d.CreateFromBytes(data)
}

// Create create deployment from yaml file, alias to "CreateFromBytes"
func (d *Deployment) Create(path string) (*appsv1.Deployment, error) {
	return d.CreateFromFile(path)
}

// UpdateFromRaw update deployment from map[string]interface{}
func (d *Deployment) UpdateFromRaw(raw map[string]interface{}) (*appsv1.Deployment, error) {
	deploy := &appsv1.Deployment{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, deploy)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(deploy.Namespace) != 0 {
		namespace = deploy.Namespace
	} else {
		namespace = d.namespace
	}

	return d.clientset.AppsV1().Deployments(namespace).Update(d.ctx, deploy, d.Options.UpdateOptions)
}

// UpdateFromBytes update deploy from bytes
func (d *Deployment) UpdateFromBytes(data []byte) (*appsv1.Deployment, error) {
	deployJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	deploy := &appsv1.Deployment{}
	err = json.Unmarshal(deployJson, deploy)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(deploy.Namespace) != 0 {
		namespace = deploy.Namespace
	} else {
		namespace = d.namespace
	}

	return d.clientset.AppsV1().Deployments(namespace).Update(d.ctx, deploy, d.Options.UpdateOptions)
}

// UpdateFromFile update deployment from yaml file
func (d *Deployment) UpdateFromFile(path string) (*appsv1.Deployment, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return d.UpdateFromBytes(data)
}

// Update update deployment from yaml file, alias to "UpdateFromFile"
func (d *Deployment) Update(path string) (deploy *appsv1.Deployment, err error) {
	return d.UpdateFromFile(path)
}

// ApplyFromRaw apply deployment from map[string]interface{}
func (d *Deployment) ApplyFromRaw(raw map[string]interface{}) (*appsv1.Deployment, error) {
	deploy := &appsv1.Deployment{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, deploy)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(deploy.Namespace) != 0 {
		namespace = deploy.Namespace
	} else {
		namespace = d.namespace
	}

	deploy, err = d.clientset.AppsV1().Deployments(namespace).Create(d.ctx, deploy, d.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		deploy, err = d.clientset.AppsV1().Deployments(namespace).Update(d.ctx, deploy, d.Options.UpdateOptions)
	}
	return deploy, err
}

// ApplyFromBytes pply deployment from bytes
func (d *Deployment) ApplyFromBytes(data []byte) (deploy *appsv1.Deployment, err error) {
	deploy, err = d.CreateFromBytes(data)
	if k8serrors.IsAlreadyExists(err) {
		deploy, err = d.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply deployment from yaml file
func (d *Deployment) ApplyFromFile(path string) (deploy *appsv1.Deployment, err error) {
	deploy, err = d.CreateFromFile(path)
	if k8serrors.IsAlreadyExists(err) { // if deployment already exist, update it.
		deploy, err = d.UpdateFromFile(path)
	}
	return
}

// ApplyFromFile apply deployment from yaml file, alias to "ApplyFromFile"
func (d *Deployment) Apply(path string) (*appsv1.Deployment, error) {
	return d.ApplyFromFile(path)
}

func (d *Deployment) Apply2(path string) (deploy *appsv1.Deployment, err error) {
	var (
		data            []byte
		deployJson      []byte
		namespace       string
		bufferSize      = 500
		unstructuredMap map[string]interface{}
		unstructuredObj = &unstructured.Unstructured{}
	)
	deploy = &appsv1.Deployment{}
	if data, err = ioutil.ReadFile(path); err != nil {
		return
	}
	if deployJson, err = yaml.ToJSON(data); err != nil {
		return
	}
	if err = json.Unmarshal(deployJson, deploy); err != nil {
		return
	}
	if len(deploy.Namespace) != 0 {
		namespace = deploy.Namespace
	} else {
		namespace = d.namespace
	}
	_ = namespace
	// NewYAMLOrJSONDecoder returns a decoder that will process YAML documents
	// or JSON documents from the given reader as a stream. bufferSize determines
	// how far into the stream the decoder will look to figure out whether this
	// is a JSON stream (has whitespace followed by an open brace).
	// yaml documents io.Reader  --> yaml decoder util/yaml.YAMLOrJSONDecoder
	decoder := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), bufferSize)
	for {
		// RawExtension is used to hold extensions in external versions.
		var rawObject runtime.RawExtension
		// Decode reads a YAML document as JSON from the stream or returns an error.
		// The decoding rules match json.Unmarshal, not yaml.Unmarshal.
		// 用来判断文件内容是不是 yaml 格式. 如果 decoder.Decode 返回了错误, 说明文件内容
		// 不是 yaml 格式的. 如果返回 nil, 说明文件内容是 yaml 格式
		// yaml decoder util/yaml.YAMLOrJSONDecoder --> json serializer runtime.Serializer
		if err := decoder.Decode(&rawObject); err != nil {
			break
		}
		if len(rawObject.Raw) == 0 {
			// if the yaml object is empty just continue to the next one
			continue
		}
		// NewDecodingSerializer adds YAML decoding support to a serializer that supports JSON.
		// json serializer runtime.Serializer --> runtime.Object, *schema.GroupVersionKind
		object, gvk, err := serializeryaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObject.Raw, nil, nil)
		if err != nil {
			log.Error("NewDecodingSerializer error")
			log.Error(err)
			return nil, err
		}
		// runtime.Object --> map[string]interface{}
		unstructuredMap, err = runtime.DefaultUnstructuredConverter.ToUnstructured(object)
		if err != nil {
			return nil, err
		}
		// map[string]interface{} --> unstructured.Unstructured
		unstructuredObj = &unstructured.Unstructured{Object: unstructuredMap}

		// GetAPIGroupResources uses the provided discovery client to gather
		// discovery information and populate a slice of APIGroupResources.
		// DiscoveryInterface / DiscoveryClient --> []*APIGroupResources
		apiGroupResources, err := restmapper.GetAPIGroupResources(d.clientset.Discovery())
		if err != nil {
			log.Error("GetAPIGroupResources error")
			log.Error(err)
			return nil, err
		}

		// NewDiscoveryRESTMapper returns a PriorityRESTMapper based on the discovered
		// groups and resources passed in.
		// []*APIGroupResources --> meta.RESTMapper
		restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)

		// meta.RESTMapper -> meta.RESTMapping
		// RESTMapping identifies a preferred resource mapping for the provided group kind.
		restMapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			log.Error("RESTMapping error")
			log.Error(err)
			return nil, err
		}

		var dri dynamic.ResourceInterface
		// Scope contains the information needed to deal with REST Resources that are in a resource hierarchy
		// meta.RESTMapping.Resource --> shcema.GropuVersionResource
		if restMapping.Scope.Name() == meta.RESTScopeNameNamespace { // meta.RESTScopeNameNamespace is a const, and value is default
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace("default")
			}
			dri = d.dynamicClient.Resource(restMapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = d.dynamicClient.Resource(restMapping.Resource)
		}
		_, err = dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{})
		if k8serrors.IsAlreadyExists(err) {
			_, err = dri.Update(context.Background(), unstructuredObj, metav1.UpdateOptions{})
		}
		if err != nil {
			log.Error("DynamicResourceInterface Apply error")
			log.Error(err)
			return nil, err
		}
	}
	//if err != io.EOF {
	//    log.Error("not io.EOF")
	//    log.Error(err)
	//    return nil, err
	//}

	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), deploy); err != nil {
		log.Error("FromUnstructured error")
		log.Error(err)
		return nil, err
	}
	return deploy, nil
}

// DeleteFromBytes delete deploy from bytes
func (d *Deployment) DeleteFromBytes(data []byte) error {
	deployJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	deploy := &appsv1.Deployment{}
	err = json.Unmarshal(deployJson, deploy)
	if err != nil {
		return err
	}

	var namespace string
	if len(deploy.Namespace) != 0 {
		namespace = deploy.Namespace
	} else {
		namespace = d.namespace
	}

	return d.WithNamespace(namespace).DeleteByName(deploy.Name)
}

// DeleteFromFile delete deployment from yaml file
func (d *Deployment) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return d.DeleteFromBytes(data)
}

// DeleteByName delete deployment by name
func (d *Deployment) DeleteByName(name string) (err error) {
	return d.clientset.AppsV1().Deployments(d.namespace).Delete(d.ctx, name, d.Options.DeleteOptions)
}

// Delete delete deployment by name, alias to "DeleteByName"
func (d *Deployment) Delete(name string) error {
	return d.DeleteByName(name)
}

// GetFromBytes get deployment from bytes
func (d *Deployment) GetFromBytes(data []byte) (*appsv1.Deployment, error) {

	deployJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	deploy := &appsv1.Deployment{}
	err = json.Unmarshal(deployJson, deploy)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(deploy.Namespace) != 0 {
		namespace = deploy.Namespace
	} else {
		namespace = d.namespace
	}

	return d.WithNamespace(namespace).GetByName(deploy.Name)
}

// GetFromFile get deployment from yaml file
func (d *Deployment) GetFromFile(path string) (*appsv1.Deployment, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return d.GetFromBytes(data)
}

// GetByName get deployment by name
func (d *Deployment) GetByName(name string) (*appsv1.Deployment, error) {
	return d.clientset.AppsV1().Deployments(d.namespace).Get(d.ctx, name, d.Options.GetOptions)
}

// Get get deployment by name, alias to "GetByName"
func (d *Deployment) Get(name string) (*appsv1.Deployment, error) {
	return d.clientset.AppsV1().Deployments(d.namespace).Get(d.ctx, name, d.Options.GetOptions)
}

// ListByLabel list deployments by labels
func (d *Deployment) ListByLabel(labels string) (*appsv1.DeploymentList, error) {
	//d.Options.ListOptions.LabelSelector = labelSelector
	listOptions := d.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return d.clientset.AppsV1().Deployments(d.namespace).List(d.ctx, *listOptions)
}

// List list deployments by labels, alias to "ListByLabel"
func (d *Deployment) List(label string) (*appsv1.DeploymentList, error) {
	return d.ListByLabel(label)
}

//// ListByNode list deployments by k8s node name
//// deployment not support list by k8s node name
//func (d *Deployment) ListByNode(name string) (*appsv1.DeploymentList, error) {
//    // ParseSelector takes a string representing a selector and returns an
//    // object suitable for matching, or an error.
//    fieldSelector, err := fields.ParseSelector(fmt.Sprintf("spec.nodeName=%s", name))
//    if err != nil {
//        return nil, err
//    }
//    listOptions := d.Options.ListOptions.DeepCopy()
//    listOptions.FieldSelector = fieldSelector.String()

//    return d.clientset.AppsV1().Deployments(metav1.NamespaceAll).List(d.ctx, *listOptions)
//}

// ListByNamespace list deployments in the specified namespace
func (d *Deployment) ListByNamespace(namespace string) (*appsv1.DeploymentList, error) {
	return d.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all deployments in the k8s cluster
func (d *Deployment) ListAll() (*appsv1.DeploymentList, error) {
	return d.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// GetPods get deployment all pods
func (d *Deployment) GetPods(name string) (podList []string, err error) {
	// 先检查 deployment 是否就绪
	err = d.WaitReady(name, true)
	if err != nil {
		return
	}
	if !d.IsReady(name) {
		err = fmt.Errorf("deployment %s not ready", name)
		return
	}

	// 获取一个 appsv1.Deployment 对象
	deploy, err := d.Get(name)
	if err != nil {
		return
	}
	// 获取一个 deployment 的 spec.selector.matchLabels 字段, 用来找出 deployment 创建的 pod
	matchLabels := deploy.Spec.Selector.MatchLabels
	labelSelector := ""
	for key, value := range matchLabels {
		labelSelector = labelSelector + fmt.Sprintf("%s=%s,", key, value)
	}
	labelSelector = strings.TrimRight(labelSelector, ",")
	podObjList, err := d.clientset.CoreV1().Pods(d.namespace).List(d.ctx,
		metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return
	}
	// 获取所有 pod, 并放入到 podList 列表中
	for _, pod := range podObjList.Items {
		podList = append(podList, pod.Name)
	}
	return
}

// GetPV get deployment pv list by name
func (d *Deployment) GetPV(name string) (pvList []string, err error) {
	var (
		pvcHandler *PersistentVolumeClaim
		pvcObj     *corev1.PersistentVolumeClaim
		pvcList    []string
	)
	// 等待 deployment 就绪
	err = d.WaitReady(name, true)
	if err != nil {
		return
	}
	// 在获取 pv 之前, 先判断 deployment 是否就绪, 如果还不是则直接退出
	if !d.IsReady(name) {
		err = fmt.Errorf("deployment %s not ready", name)
		return
	}
	// 获取一个用来处理 *corev1.PersistentVolumeClaim 的处理器
	pvcHandler, err = NewPersistentVolumeClaim(d.ctx, d.namespace, d.kubeconfig)
	if err != nil {
		return
	}
	pvcList, err = d.GetPVC(name)
	if err != nil {
		return
	}
	for _, pvcName := range pvcList {
		// get *corev1.PersistentVolumeClaim
		// 通过 pvc 处理器获得 *corev1.PersistentVolumeClaim 对象
		pvcObj, err = pvcHandler.Get(pvcName)
		if err != nil {
			return
		}
		// pvcObj.Spec.VolumeName 的值就是 pvc 中绑定的 pv 的名字
		pvList = append(pvList, pvcObj.Spec.VolumeName)
	}
	return
}

// GetPVC get deployment pvc list by name
func (d *Deployment) GetPVC(name string) (pvcList []string, err error) {
	// 等待 deployment 就绪
	err = d.WaitReady(name, true)
	if err != nil {
		return
	}
	// 在获取 pvc 之前, 先判断 deployment 是否就绪, 如果还不是则直接退出
	if !d.IsReady(name) {
		err = fmt.Errorf("deployment %s not ready", name)
		return
	}

	deploy, err := d.Get(name)
	if err != nil {
		return
	}
	for _, volume := range deploy.Spec.Template.Spec.Volumes {
		// 有些 volume.PersistentVolumeClaim 是不存在的, 其值默认是 nil 如果不加以判断就直接获取
		// volume.PersistentVolumeClaim.ClaimName, 就操作了非法地址, 程序会直接报错.
		if volume.PersistentVolumeClaim != nil {
			pvcList = append(pvcList, volume.PersistentVolumeClaim.ClaimName)
		}
	}
	return
}

// IsReady check if the deployment is ready
func (d *Deployment) IsReady(name string) bool {
	// 获取 *appsv1.Deployment
	deploy, err := d.Get(name)
	if err != nil {
		return false
	}
	// 必须 Type=Available 和 Status=True 才能算 Deployment 就绪了
	for _, cond := range deploy.Status.Conditions {
		if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// WaitReady wait for the deployment to be in the ready state
func (d *Deployment) WaitReady(name string, check bool) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
	)
	// 在 watch deployment 之前先判断 deployment 是否就绪, 如果 deployment 已经就绪了,就没必要 watch 了.
	if d.IsReady(name) {
		return
	}
	// 是否检查 deployment 是否存在
	if check {
		// 检查 deploymen 是否存在,如果存在则不用 watch
		if _, err = d.Get(name); err != nil {
			return
		}
	}
	for {
		// 开始监听 deployment 事件,
		// 1. 如果监听到了 modified 事件, 就检查下 deployment 的状态.
		//    如果 conditions.Type == appsv1.DeploymentAvailable 并且 conditions.Status == corev1.ConditionTrue
		//    说明 deployment 已经准备好了.
		// 2. 如果监听到 watch.Deleted 事件, 说明 deployment 已经删除了, 不需要再监听了
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: d.namespace})
		listOptions.TimeoutSeconds = &timeout
		watcher, err = d.clientset.AppsV1().Deployments(d.namespace).Watch(d.ctx, listOptions)
		if err != nil {
			return
		}
		// 连接 kubernetes 是有通过 http/https 方式连接的, 有一个 keepalived 的时间
		// 时间一到, 就会断开 kubernetes 的连接, 此时  watch.ResultChan 通道就会关闭.
		// 所以说, 这个方法 WaitReady 等待 deployment 处于就绪状态的最长等待时间就是
		// 连接 kubernetes 的 keepalive 时间. 好像是10分钟
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Modified:
				if d.IsReady(name) {
					watcher.Stop()
					return
				}
			case watch.Deleted: // deployment 已经删除了, 退出监听
				watcher.Stop()
				return fmt.Errorf("%s deleted", name)
			case watch.Bookmark:
				log.Debug("watch deployment: bookmark.")
			case watch.Error:
				log.Debug("watch deployment: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch deployment: reconnect to kubernetes.")
		watcher.Stop()
	}
}

// WatchByName watch deployment by name
func (d *Deployment) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)

	// if event channel is closed, it means the server has closed the connection,
	// reconnect to kubernetes.
	for {
		//watcher, err := clientset.AppsV1().Deployments(namespace).Watch(ctx,
		//    metav1.SingleObject(metav1.ObjectMeta{Name: "dep", Namespace: namespace}))
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: d.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = d.clientset.AppsV1().Deployments(d.namespace).Watch(d.ctx, listOptions); err != nil {
			return
		}
		if _, err = d.Get(name); err != nil {
			isExist = false // deployment not exist
		} else {
			isExist = true // deployment exist
		}
		//for {
		//    // kubernetes retains the resource event history, which includes this
		//    // initial event, so that when our program first start, we are automatically
		//    // notified of the deployment existence and current state.
		//    event, isOpen := <-watcher.ResultChan()

		//    if isOpen {
		//        switch event.Type {
		//        case watch.Added:
		//            // if deployment exist, skip deployment history add event.
		//            if !isExist {
		//                addFunc()
		//            }
		//            isExist = true
		//        case watch.Modified:
		//            modifyFunc()
		//            isExist = true
		//        case watch.Deleted:
		//            deleteFunc()
		//            isExist = false
		//        //case watch.Bookmark:
		//        //    log.Debug("bookmark")
		//        //case watch.Error:
		//        //    log.Error("error")
		//        default: // do nothing
		//        }
		//    } else {
		//        // If event channel is closed, it means the server has closed the connection
		//        log.Debug("reconnect to kubernetes.")
		//        break
		//    }
		//}
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
				log.Debug("watch deployment: bookmark.")
			case watch.Error:
				log.Debug("watch deployment: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch deployment: reconnect to kubernetes")
		watcher.Stop()
	}
}

// WatchByLabel watch deployment by label
func (d *Deployment) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher    watch.Interface
		timeout    = int64(0)
		isExist    bool
		deployList *appsv1.DeploymentList
	)
	// if event channel is closed, it means the server has closed the connection,
	// reconnect to kubernetes.
	for {
		//watcher, err := clientset.AppsV1().Deployments(namespace).Watch(ctx,
		//    metav1.SingleObject(metav1.ObjectMeta{Name: "dep", Namespace: namespace}))
		// 这个 timeout 一定要设置为 0, 否则 watcher 就会中断
		if watcher, err = d.clientset.AppsV1().Deployments(d.namespace).Watch(d.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if deployList, err = d.List(labelSelector); err != nil {
			return
		}
		if len(deployList.Items) == 0 {
			isExist = false // deployment not exist
		} else {
			isExist = true // deployment exist
		}
		//for {
		//    // kubernetes retains the resource event history, which includes this
		//    // initial event, so that when our program first start, we are automatically
		//    // notified of the deployment existence and current state.
		//    event, isOpen := <-watcher.ResultChan()

		//    if isOpen {
		//        switch event.Type {
		//        case watch.Added:
		//            // if deployment exist, skip deployment history add event.
		//            if !isExist {
		//                addFunc(x)
		//            }
		//            isExist = true
		//        case watch.Modified:
		//            modifyFunc(x)
		//            isExist = true
		//        case watch.Deleted:
		//            deleteFunc(x)
		//            isExist = false
		//        //case watch.Bookmark:
		//        //    log.Debug("bookmark")
		//        //case watch.Error:
		//        //    log.Error("error")
		//        default: // do nothing
		//        }
		//    } else {
		//        // If event channel is closed, it means the server has closed the connection
		//        log.Debug("reconnect to kubernetes.")
		//        break
		//    }
		//}
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
				log.Debug("watch deployment: bookmark.")
			case watch.Error:
				log.Debug("watch deployment: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch deployment: reconnect to kubernetes")
		watcher.Stop()
	}
}

// Watch watch deployment by label, alias to "WatchByLabel"
func (d *Deployment) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return d.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
