package clientset

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Pod struct {
	namespace string
	limit     int64
	timeout   int64
	ctx       context.Context
	client    *kubernetes.Clientset

	sync.Mutex
}

// new a `Pod` instance from kubeconfig or in-cluster config
func NewPod(ctx context.Context, namespace, kubeconfig string) (pod *Pod, err error) {
	var (
		config *rest.Config
		client *kubernetes.Clientset
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

	// create the clientset
	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	pod.namespace = namespace
	pod.limit = 100
	pod.timeout = 10
	pod.ctx = ctx
	pod.client = client

	return
}
func (p *Pod) SetLimit(limit int64) {
	p.Lock()
	defer p.Unlock()
	p.limit = limit
}
func (p *Pod) SetTimeout(timeout int64) {
	p.Lock()
	defer p.Unlock()
	p.timeout = timeout
}

// create pod from file
func (p *Pod) Create(filepath string) (pod *corev1.Pod, err error) {
	var (
		podYaml []byte
		podJson []byte
	)
	pod = &corev1.Pod{}
	if podYaml, err = ioutil.ReadFile(filepath); err != nil {
		return
	}
	if podJson, err = yaml.ToJSON(podYaml); err != nil {
		return
	}
	if err = json.Unmarshal(podJson, pod); err != nil {
		return
	}
	pod, err = p.client.CoreV1().Pods(p.namespace).Create(p.ctx, pod, metav1.CreateOptions{})
	return
}

// update pod from file
func (p *Pod) Update(filepath string) (pod *corev1.Pod, err error) {
	var (
		podYaml []byte
		podJson []byte
	)
	pod = &corev1.Pod{}
	if podYaml, err = ioutil.ReadFile(filepath); err != nil {
		return
	}
	if podJson, err = yaml.ToJSON(podYaml); err != nil {
		return
	}
	if err = json.Unmarshal(podJson, pod); err != nil {
		return
	}
	pod, err = p.client.CoreV1().Pods(p.namespace).Update(p.ctx, pod, metav1.UpdateOptions{})
	return
}

// apply pod from file
func (p *Pod) Apply(filepath string) (pod *corev1.Pod, err error) {
	pod, err = p.Create(filepath)
	if errors.IsAlreadyExists(err) {
		pod, err = p.Update(filepath)
	}
	return
}

// delete pod by name
func (p *Pod) Delete(name string, forceDelete bool) (err error) {
	var gracePeriodSeconds int64
	if forceDelete {
		gracePeriodSeconds = 0
		err = p.client.CoreV1().Pods(p.namespace).Delete(p.ctx,
			name, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds})
		return
	}
	err = p.client.CoreV1().Pods(p.namespace).Delete(p.ctx, name, metav1.DeleteOptions{})
	return
}

// get pod by name
func (p *Pod) Get(name string) (pod *corev1.Pod, err error) {
	pod, err = p.client.CoreV1().Pods(p.namespace).Get(p.ctx, name, metav1.GetOptions{})
	return
}

// list pods by labelSelector
func (p *Pod) List(labelSelector string) (podList *corev1.PodList, err error) {
	podList, err = p.client.CoreV1().Pods(p.namespace).List(p.ctx,
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &p.timeout, Limit: p.limit})
	return
}

// watch pods by labelSelector
func (p *Pod) Watch(labelSelector string, addFunc, modifyFunc, deleteFunc func()) (err error) {
	var (
		watcher watch.Interface
		podList *corev1.PodList
		timeout = int64(0)
		isExist bool
	)
	for {
		if watcher, err = p.client.CoreV1().Pods(p.namespace).Watch(p.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout, Limit: p.limit}); err != nil {
			logrus.Error(err)
			return
		}
		if podList, err = p.List(labelSelector); err != nil {
			logrus.Error(err)
			return
		}
		if len(podList.Items) == 0 {
			isExist = false // pod not exist
		} else {
			isExist = true // pod exist
		}
		for {
			event, isOpen := <-watcher.ResultChan()
			if isOpen {
				switch event.Type {
				case watch.Added:
					if !isExist {
						addFunc()
					}
					isExist = true
				case watch.Modified:
					modifyFunc()
					isExist = true
				case watch.Deleted:
					deleteFunc()
					isExist = false
				default:
				}
			} else {
				// If event channel is closed, it means the server has closed the connection
				logrus.Info("reconnect to kube-apiserver")
				break
			}
		}
	}
}
