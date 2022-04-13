package k8s

// TODO:
// 1. 加了 Options 参数之后可能要修改的东西: List, Watch, WaitReady
// 2. 精简代码, 比如返回值
import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"

	//"k8s.io/client-go/deprecated/scheme"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type Container struct {
	Name  string
	Image string
}
type PodController struct {
	//APIVersion        string            `json:"apiVersion"`
	//Kind              string            `json:"kind"`
	//Name              string            `json:"name"`
	//UID               string            `json:"uid"`
	//Controller        bool              `json:"controller"`
	//BlockOwnerDeletion    bool `json:"blockOwnerDeletion"`
	Labels            map[string]string `json:"labels"`
	Ready             string            `json:"ready"`
	Images            []string          `json:"images"`
	CreationTimestamp metav1.Time       `json:"creationTimestamp"`

	metav1.OwnerReference `json:"ownerReference"`
}
type Pod struct {
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

// new a Pod handler from kubeconfig or in-cluster config
func NewPod(ctx context.Context, namespace, kubeconfig string) (pod *Pod, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
	)
	pod = &Pod{}

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
	//config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
	config.NegotiatedSerializer = scheme.Codecs
	//config.UserAgent = rest.DefaultKubernetesUserAgent()
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
	pod.kubeconfig = kubeconfig
	pod.namespace = namespace

	pod.ctx = ctx
	pod.config = config
	pod.restClient = restClient
	pod.clientset = clientset
	pod.dynamicClient = dynamicClient
	pod.discoveryClient = discoveryClient

	pod.Options = &HandlerOptions{}

	return
}
func (p *Pod) Namespace() string {
	return p.namespace
}
func (in *Pod) DeepCopy() *Pod {
	out := new(Pod)

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

func (p *Pod) setNamespace(namespace string) {
	p.Lock()
	defer p.Unlock()
	p.namespace = namespace
}
func (p *Pod) WithNamespace(namespace string) *Pod {
	pod := p.DeepCopy()
	pod.setNamespace(namespace)
	return pod
}
func (p *Pod) WithDryRun() *Pod {
	pod := p.DeepCopy()
	pod.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	pod.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	pod.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	pod.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	pod.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return pod
}
func (p *Pod) SetLimit(limit int64) {
	p.Lock()
	defer p.Unlock()
	p.Options.ListOptions.Limit = limit
}
func (p *Pod) SetTimeout(timeout int64) {
	p.Lock()
	defer p.Unlock()
	p.Options.ListOptions.TimeoutSeconds = &timeout
}
func (p *Pod) SetForceDelete(force bool) {
	p.Lock()
	defer p.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		p.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		p.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// create pod from bytes
func (p *Pod) CreateFromBytes(data []byte) (*corev1.Pod, error) {
	podJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	pod := &corev1.Pod{}
	err = json.Unmarshal(podJson, pod)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(pod.Namespace) != 0 {
		namespace = pod.Namespace
	} else {
		namespace = p.namespace
	}

	return p.clientset.CoreV1().Pods(namespace).Create(p.ctx, pod, p.Options.CreateOptions)
}

// create pod from file
func (p *Pod) CreateFromFile(path string) (*corev1.Pod, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p.CreateFromBytes(data)
}

// create pod from file, alias to "CreateFromFile"
func (p *Pod) Create(path string) (*corev1.Pod, error) {
	return p.CreateFromFile(path)
}

// update pod from bytes
func (p *Pod) UpdateFromBytes(data []byte) (*corev1.Pod, error) {
	podJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	pod := &corev1.Pod{}
	err = json.Unmarshal(podJson, pod)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(pod.Namespace) != 0 {
		namespace = pod.Namespace
	} else {
		namespace = p.namespace
	}

	return p.clientset.CoreV1().Pods(namespace).Update(p.ctx, pod, p.Options.UpdateOptions)
}

// update pod from file
func (p *Pod) UpdateFromFile(path string) (*corev1.Pod, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p.UpdateFromBytes(data)
}

// update pod from file, alias to "UpdateFromFile"
func (p *Pod) Update(path string) (*corev1.Pod, error) {
	return p.UpdateFromFile(path)
}

// apply pod from bytes
func (p *Pod) ApplyFromBytes(data []byte) (pod *corev1.Pod, err error) {
	pod, err = p.CreateFromBytes(data)
	if k8serrors.IsAlreadyExists(err) {
		log.Debug(err)
		pod, err = p.UpdateFromBytes(data)
	}
	return
}

// apply pod from file
func (p *Pod) ApplyFromFile(path string) (pod *corev1.Pod, err error) {
	pod, err = p.CreateFromFile(path)
	if k8serrors.IsAlreadyExists(err) {
		pod, err = p.UpdateFromFile(path)
	}
	return
}

// apply pod from file, alias to "ApplyFromFile"
func (p *Pod) Apply(path string) (*corev1.Pod, error) {
	return p.ApplyFromFile(path)
}

// delete pod from bytes
func (p *Pod) DeleteFromBytes(data []byte) error {
	podJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	pod := &corev1.Pod{}
	err = json.Unmarshal(podJson, pod)
	if err != nil {
		return err
	}

	var namespace string
	if len(pod.Namespace) != 0 {
		namespace = pod.Namespace
	} else {
		namespace = p.namespace
	}

	return p.WithNamespace(namespace).DeleteByName(pod.Name)
}

// delete pod from file
func (p *Pod) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return p.DeleteFromBytes(data)
}

// delete pod by name
func (p *Pod) DeleteByName(name string) error {
	return p.clientset.CoreV1().Pods(p.namespace).Delete(p.ctx, name, p.Options.DeleteOptions)
}

// delete pod by name, alias to "DeleteByName"
func (p *Pod) Delete(name string) error {
	return p.DeleteByName(name)
}

// get pod from bytes
func (p *Pod) GetFromBytes(data []byte) (*corev1.Pod, error) {
	podJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	pod := &corev1.Pod{}
	if err = json.Unmarshal(podJson, pod); err != nil {
		return nil, err
	}

	var namespace string
	if len(pod.Namespace) != 0 {
		namespace = pod.Namespace
	} else {
		namespace = p.namespace
	}

	return p.WithNamespace(namespace).GetByName(pod.Name)
}

// get pod from file
func (p *Pod) GetFromFile(path string) (*corev1.Pod, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p.GetFromBytes(data)
}

// get pod by name
func (p *Pod) GetByName(name string) (*corev1.Pod, error) {
	return p.clientset.CoreV1().Pods(p.namespace).Get(p.ctx, name, p.Options.GetOptions)
}

// get pod by name
func (p *Pod) Get(name string) (pod *corev1.Pod, err error) {
	return p.GetByName(name)
}

// ListByLabel list pods by labels
func (p *Pod) ListByLabel(labels string) (*corev1.PodList, error) {
	// TODO: 合并 ListOptions
	listOptions := p.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return p.clientset.CoreV1().Pods(p.namespace).List(p.ctx, *listOptions)
}

// List list pods by labels, alias to "ListByLabel"
func (p *Pod) List(labels string) (*corev1.PodList, error) {
	return p.ListByLabel(labels)
}

// ListByNode list all pods in k8s node where the pod is running
func (p *Pod) ListByNode(name string) (*corev1.PodList, error) {
	// ParseSelector takes a string representing a selector and returns an
	// object suitable for matching, or an error.
	fieldSelector, err := fields.ParseSelector(fmt.Sprintf("spec.nodeName=%s", name))
	if err != nil {
		return nil, err
	}
	listOptions := p.Options.ListOptions.DeepCopy()
	listOptions.FieldSelector = fieldSelector.String()

	return p.clientset.CoreV1().Pods(metav1.NamespaceAll).List(p.ctx, *listOptions)
}

// ListByNamespace list all pods in the specified namespace
func (p *Pod) ListByNamespace(namespace string) (*corev1.PodList, error) {
	return p.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all pods in k8s cluster where the pod is running
func (p *Pod) ListAll() (*corev1.PodList, error) {
	return p.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// get pod ip
func (p *Pod) GetIP(name string) (podIP string, err error) {
	// 先检查 pod 是否就绪
	err = p.WaitReady(name, true)
	if err != nil {
		return
	}
	if !p.IsReady(name) {
		err = fmt.Errorf("pod %s not ready", name)
		return
	}

	// 创建一个 corev1.Pod 对象
	pod, err := p.Get(name)
	if err != nil {
		return
	}
	podIP = pod.Status.PodIP
	return
}

// get pod uuid
func (p *Pod) GetUID(name string) (uid string, err error) {
	// 先检查 pod 是否就绪
	err = p.WaitReady(name, true)
	if err != nil {
		return
	}
	if !p.IsReady(name) {
		err = fmt.Errorf("pod %s not ready", name)
		return
	}

	// 创建一个 corev1.Pod 对象
	pod, err := p.Get(name)
	if err != nil {
		return
	}
	uid = string(pod.ObjectMeta.UID)
	return
}

// get the ip addr of the node where pod is located
func (p *Pod) GetNodeIP(name string) (nodeIP string, err error) {
	// 检查 pod 是否就绪
	err = p.WaitReady(name, true)
	if err != nil {
		return
	}
	if !p.IsReady(name) {
		err = fmt.Errorf("pod %s not ready", name)
		return
	}

	// 创建一个 corev1.Pod 对象
	pod, err := p.Get(name)
	if err != nil {
		return
	}
	nodeIP = pod.Status.HostIP
	return
}

// get the name of the node where pod is located
func (p *Pod) GetNodeName(name string) (nodeName string, err error) {
	// 检查 pod 是否就绪
	err = p.WaitReady(name, true)
	if err != nil {
		return
	}
	if !p.IsReady(name) {
		err = fmt.Errorf("pod %s not ready", name)
		return
	}

	// 创建一个 corev1.Pod 对象
	pod, err := p.Get(name)
	if err != nil {
		return
	}
	nodeName = pod.Spec.NodeName
	return
}

// get the all containers in the pod
func (p *Pod) GetContainers(name string) (containerList []Container, err error) {
	// 检查 pod 是否就绪
	err = p.WaitReady(name, true)
	if err != nil {
		return nil, err
	}
	if !p.IsReady(name) {
		err = fmt.Errorf("pod %s not ready", name)
		return
	}
	// 创建一个 corev1.Pod 对象
	pod, err := p.Get(name)
	if err != nil {
		return
	}

	//for _, container := range pod.Spec.Containers {
	//    c := Container{
	//        Name:  container.Name,
	//        Image: container.Image,
	//    }
	//    containerList = append(containerList, c)
	//}

	for _, cs := range pod.Status.ContainerStatuses {
		c := Container{
			Name:  cs.Name,
			Image: cs.Image,
		}
		containerList = append(containerList, c)
	}

	return
}

// get ready containers in the pod
func (p *Pod) GetReadyContainers(name string) (containerList []Container, err error) {
	err = p.WaitReady(name, true)
	if err != nil {
		return nil, err
	}
	if !p.IsReady(name) {
		err = fmt.Errorf("pod %s not ready", name)
		return
	}
	pod, err := p.Get(name)
	if err != nil {
		return
	}

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			c := Container{
				Name:  cs.Name,
				Image: cs.Image,
			}
			containerList = append(containerList, c)
		}
	}

	return
}

// get the pod pv list by name.
func (p *Pod) GetPV(name string) (pvList []string, err error) {
	var (
		pvcHandler *PersistentVolumeClaim
		pvcObj     *corev1.PersistentVolumeClaim
		pvcList    []string
	)
	// 等待 pod 就绪
	err = p.WaitReady(name, true)
	if err != nil {
		return
	}
	// 判断 pod  是否是就绪状态, 如果不是,直接退出
	if !p.IsReady(name) {
		err = fmt.Errorf("pod %s not ready", name)
		return
	}

	// 先创建一个用来处理 PersistentVolumeClaim 的对象
	pvcHandler, err = NewPersistentVolumeClaim(p.ctx, p.namespace, p.kubeconfig)
	if err != nil {
		return
	}
	// 先获取 pvc list
	pvcList, err = p.GetPVC(name)
	if err != nil {
		return
	}
	// 循环获取 pv
	for _, pvcName := range pvcList {
		// 获取 *corev1.PersistentVolumeClaim 对象
		pvcObj, err = pvcHandler.Get(pvcName)
		pvList = append(pvList, pvcObj.Spec.VolumeName)
	}
	return
}

// get the pod pvc list by name
func (p *Pod) GetPVC(name string) (pvcList []string, err error) {
	// 等待 pod 就绪
	err = p.WaitReady(name, true)
	if err != nil {
		return
	}
	// 判断 pod 是否是就绪状态, 如果不是,直接退出
	if !p.IsReady(name) {
		err = fmt.Errorf("pod %s not ready", name)
		return
	}

	// 先获取一个 corev1.Pod 对象
	pod, err := p.Get(name)
	if err != nil {
		return
	}
	for _, volume := range pod.Spec.Volumes {
		// 要先判断 volume.PersistentVolumeClaim 是否为空, 如果不判断而直接获取
		// volume.PersistentVolumeClaim.ClaimName 相当于操纵值为 nil 的指针(空指针),
		// 程序会直接中断退出.
		if volume.PersistentVolumeClaim != nil {
			pvcList = append(pvcList, volume.PersistentVolumeClaim.ClaimName)
		}
	}

	return
}

// GetController returns a *PodController object by pod name if the controllee(pod) has a controller
func (p *Pod) GetController(name string) (*PodController, error) {
	var (
		podHandler *Pod
		stsHandler *StatefulSet
		dsHandler  *DaemonSet
		jobHandler *Job
		rsHandler  *ReplicaSet
		rcHandler  *ReplicationController
	)
	if len(name) == 0 {
		return nil, fmt.Errorf("not set the pod name")
	}
	pod, err := p.Get(name)
	if err != nil {
		return nil, err
	}
	// GetControllerOf returns a pointer to a copy of the controllerRef if controllee has a controller
	ownerRef := metav1.GetControllerOf(pod)
	if ownerRef == nil {
		return nil, fmt.Errorf("the pod %q doesn't have controller", name)
	}
	oc := PodController{OwnerReference: *ownerRef}

	// get containers image
	containers, err := p.GetContainers(name)
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		oc.Images = append(oc.Images, c.Image)
	}

	switch strings.ToLower(ownerRef.Kind) {
	case ResourceKindPod:
		var pod *corev1.Pod
		if podHandler, err = NewPod(p.ctx, p.namespace, p.kubeconfig); err != nil {
			return nil, err
		}
		if pod, err = podHandler.Get(oc.Name); err != nil {
			return nil, err
		}
		oc.Labels = pod.Labels
		rcs, _ := p.GetReadyContainers(oc.Name)
		oc.Ready = fmt.Sprintf("%d/%d", len(rcs), len(pod.Spec.Containers))
		oc.CreationTimestamp = pod.CreationTimestamp
	case ResourceKindDaemonSet:
		var ds *appsv1.DaemonSet
		if dsHandler, err = NewDaemonSet(p.ctx, p.namespace, p.kubeconfig); err != nil {
			return nil, err
		}
		if ds, err = dsHandler.Get(oc.Name); err != nil {
			return nil, err
		}
		oc.Labels = ds.Labels
		oc.Ready = fmt.Sprintf("%d/%d", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
		oc.CreationTimestamp = ds.CreationTimestamp
	case ResourceKindStatefulSet:
		var sts *appsv1.StatefulSet
		if stsHandler, err = NewStatefulSet(p.ctx, p.namespace, p.kubeconfig); err != nil {
			return nil, err
		}
		if sts, err = stsHandler.Get(oc.Name); err != nil {
			return nil, err
		}
		oc.Labels = sts.Labels
		oc.Ready = fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, sts.Status.Replicas)
		oc.CreationTimestamp = sts.CreationTimestamp
	case ResourceKindJob:
		var job *batchv1.Job
		if jobHandler, err = NewJob(p.ctx, p.namespace, p.kubeconfig); err != nil {
			return nil, err
		}
		if job, err = jobHandler.Get(oc.Name); err != nil {
			return nil, err
		}
		oc.Labels = job.Labels
		oc.Ready = fmt.Sprintf("%d/%d", job.Status.Succeeded, *job.Spec.Completions)
		oc.CreationTimestamp = job.CreationTimestamp
	case ResourceKindReplicaSet:
		var rs *appsv1.ReplicaSet
		if rsHandler, err = NewReplicaSet(p.ctx, p.namespace, p.kubeconfig); err != nil {
			return nil, err
		}
		if rs, err = rsHandler.Get(oc.Name); err != nil {
			return nil, err
		}
		oc.Labels = rs.Labels
		oc.Ready = fmt.Sprintf("%d/%d", rs.Status.ReadyReplicas, rs.Status.Replicas)
		oc.CreationTimestamp = rs.CreationTimestamp
	case ResourceKindReplicationController:
		var rc *corev1.ReplicationController
		if rcHandler, err = NewReplicationController(p.ctx, p.namespace, p.kubeconfig); err != nil {
			return nil, err
		}
		if rc, err = rcHandler.Get(oc.Name); err != nil {
			return nil, err
		}
		oc.Labels = rc.Labels
		oc.Ready = fmt.Sprintf("%d/%d", rc.Status.ReadyReplicas, rc.Status.Replicas)
		oc.CreationTimestamp = rc.CreationTimestamp
	default:
		return nil, fmt.Errorf("unknown reference kind: %s", ownerRef.Kind)
	}
	return &oc, nil
}

// check if the pod is ready
func (p *Pod) IsReady(name string) bool {
	// 获取 *corev1.Pod 对象
	pod, err := p.Get(name)
	if err != nil {
		return false
	}
	// 必须要 type=Ready 和 Status=True 才能算 Pod 已经就绪
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// wait for the pod to be in the ready status
func (p *Pod) WaitReady(name string, check bool) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
	)
	// 在 watch 之前就先判断 pod 是否就绪, 如果就绪了就没必要 watch 了
	if p.IsReady(name) {
		return
	}
	// 是否判断 pod 是否存在
	if check {
		if _, err = p.Get(name); err != nil {
			return
		}
	}
	for {
		// pod 没有就绪, 那么就开始监听 pod 的事件
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: p.namespace})
		listOptions.TimeoutSeconds = &timeout
		watcher, err = p.clientset.CoreV1().Pods(p.namespace).Watch(p.ctx, listOptions)
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Modified:
				if p.IsReady(name) {
					watcher.Stop()
					return
				}
			case watch.Deleted:
				watcher.Stop()
				return fmt.Errorf("%s deleted", name)
			case watch.Bookmark:
				log.Debug("watch pod: bookmark")
			case watch.Error:
				log.Debug("watch pod: error")
			}
		}
		// watcher 因为 keepalive 超时断开了连接, 关闭了 channel
		log.Debug("watch pod: reconnect to kubernetes")
		watcher.Stop()
	}
}

// wait for the pod to be in the ready status
func (p *Pod) WaitReady2(name string, check bool) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		started = make(chan struct{}, 1)
	)
	// 在 watch 之前就先判断 pod 是否就绪, 如果就绪了就没必要 watch 了
	if p.IsReady(name) {
		return
	}

	ctxCheck, cancelCheck := context.WithCancel(p.ctx)
	ctxWatch, cancelWatch := context.WithCancel(p.ctx)
	defer cancelCheck()
	defer cancelWatch()

	// 开启一个 goroutine 来监控 pod 是否存在, 如果不存在调用 cancelWatch, 取消 waitReady
	if check {
		go func(context.Context) {
			// 等待 waitReady 开始
			for {
				select {
				case <-started:
					goto THERE
				}
			}
		THERE:
			for {
				if _, err = p.Get(name); err != nil {
					cancelWatch() // 如果发现要监控的对象不存在, 则调用 cancelWatch 取消 waitReady
					return
				}
				time.Sleep(time.Second)
			}
		}(ctxCheck)
	}

	go func(ctx context.Context) {
		// 发送一个信号给 check, 告诉它我已经开始了
		started <- struct{}{}
		for {
			listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: p.namespace})
			listOptions.TimeoutSeconds = &timeout
			watcher, err = p.clientset.CoreV1().Pods(p.namespace).Watch(p.ctx, listOptions)
			for event := range watcher.ResultChan() {
				switch event.Type {
				case watch.Modified:
					if p.IsReady(name) {
						watcher.Stop()
						cancelCheck() // 我已经完成 waitReady 了, 调用 cancelCheck 来取消 check
						return
					}
				case watch.Deleted:
					watcher.Stop()
					// 没必要这个 err 了, 监控 pod 是否存在的 goroutine 会设置一个 err
					//err = fmt.Errorf("%s deleted", name)
					cancelCheck() // 我已经完成 waitReady 了, 调用 cancelCheck 来取消 check
					return
				case watch.Bookmark:
					log.Debug("watch pod: bookmark")
				case watch.Error:
					log.Debug("watch pod: error")
				}
			}
			// watcher 因为 keepalive 超时断开了连接, 关闭了 channel
			log.Debug("watch pod: reconnect to kubernetes")
			watcher.Stop()
		}
	}(ctxWatch)

	return
}

// watch pods by labelSelector
func (p *Pod) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: p.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = p.clientset.CoreV1().Pods(p.namespace).Watch(p.ctx, listOptions); err != nil {
			return
		}
		if _, err = p.Get(name); err != nil {
			isExist = false // pod not exist
		} else {
			isExist = true // pod exist
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
				log.Debug("watch pod: bookmark")
			case watch.Error:
				log.Debug("watch pod: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch pod: reconnect to kubernetes")
		watcher.Stop()
	}
}

// watch pods by labelSelector
func (p *Pod) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		podList *corev1.PodList
		timeout = int64(0)
		isExist bool
	)
	for {
		if watcher, err = p.clientset.CoreV1().Pods(p.namespace).Watch(p.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if podList, err = p.List(labelSelector); err != nil {
			return
		}
		if len(podList.Items) == 0 {
			isExist = false // pod not exist
		} else {
			isExist = true // pod exist
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
				log.Debug("watch pod: bookmark")
			case watch.Error:
				log.Debug("watch pod: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch pod: reconnect to kubernetes")
		watcher.Stop()
	}
}

// watch pods by name, alias to "WatchByName"
func (p *Pod) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return p.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}

//type PodExecOptions struct {
//    metav1.TypeMeta `json:",inline"`
//    Stdin bool `json:"stdin,omitempty" protobuf:"varint,1,opt,name=stdin"`
//    Stdout bool `json:"stdout,omitempty" protobuf:"varint,2,opt,name=stdout"`
//    Stderr bool `json:"stderr,omitempty" protobuf:"varint,3,opt,name=stderr"`
//    TTY bool `json:"tty,omitempty" protobuf:"varint,4,opt,name=tty"`
//    Container string `json:"container,omitempty" protobuf:"bytes,5,opt,name=container"`
//    Command []string `json:"command" protobuf:"bytes,6,rep,name=command"`
//}
//type StreamOptions struct {
//    Stdin             io.Reader
//    Stdout            io.Writer
//    Stderr            io.Writer
//    Tty               bool
//    TerminalSizeQueue TerminalSizeQueue
//}

// executing remote processes.
// ref:
//    https://miminar.fedorapeople.org/_preview/openshift-enterprise/registry-redeploy/go_client/executing_remote_processes.html
//    https://stackoverflow.com/questions/43314689/example-of-exec-in-k8ss-pod-by-using-go-client
//    https://github.com/kubernetes/kubernetes/blob/v1.6.1/test/e2e/framework/exec_util.go
func (p *Pod) Execute(podName, containerName string, command []string) (err error) {
	// wait pod to be ready
	err = p.WaitReady(podName, true)
	if err != nil {
		return
	}
	// 判断 pod  是否是就绪状态, 如果不是,直接退出
	if !p.IsReady(podName) {
		err = fmt.Errorf("pod %s not ready", podName)
		return
	}

	// get corev1.Pod
	pod, err := p.Get(podName)
	if err != nil {
		return
	}

	// if containerName is empty, the default containerName default is the name of the
	// first container in the pod.
	if len(containerName) == 0 {
		containerName = pod.Spec.Containers[0].Name
	}

	// Prepare the API URL used to execute another process within the Pod.  In
	// this case, we'll run a remote shell.
	req := p.restClient.Post().
		Namespace(p.namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(p.config, "POST", req.URL())
	if err != nil {
		return
	}

	// Put the terminal into raw mode to prevent it echoing characters twice
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		return
	}
	defer terminal.Restore(0, oldState)

	// Connect the process  std(in,out,err) to the remote shell process.
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    true,
	})
	if err != nil {
		return
	}

	return
}
