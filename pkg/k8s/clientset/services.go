package clientset

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Service struct {
	namespace string
	limit     int64
	timeout   int64
	ctx       context.Context
	client    *kubernetes.Clientset

	sync.Mutex
}

func NewService(ctx context.Context, namespace, kubeconfig string) (service *Service, err error) {
	var (
		config *rest.Config
		client *kubernetes.Clientset
	)
	service = &Service{}

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

	service.namespace = namespace
	service.limit = 100
	service.timeout = 10
	service.ctx = ctx
	service.client = client

	return
}
func (s *Service) SetLimit(limit int64) {
	s.Lock()
	defer s.Unlock()
	s.limit = limit
}
func (s *Service) SetTimeout(timeout int64) {
	s.Lock()
	defer s.Unlock()
	s.timeout = timeout
}

// create service from file
func (s *Service) Create(filepath string) (service *corev1.Service, err error) {
	var (
		serviceYaml []byte
		serviceJson []byte
	)
	service = &corev1.Service{}
	if serviceYaml, err = ioutil.ReadFile(filepath); err != nil {
		return
	}
	if serviceJson, err = yaml.ToJSON(serviceYaml); err != nil {
		return
	}
	if err = json.Unmarshal(serviceJson, service); err != nil {
		return
	}
	service, err = s.client.CoreV1().Services(s.namespace).Create(s.ctx, service, metav1.CreateOptions{})
	return
}

// update service from file
func (s *Service) Update(filepath string) (service *corev1.Service, err error) {
	var (
		serviceYaml []byte
		serviceJson []byte
	)
	service = &corev1.Service{}
	if serviceYaml, err = ioutil.ReadFile(filepath); err != nil {
		return
	}
	if serviceJson, err = yaml.ToJSON(serviceYaml); err != nil {
		return
	}
	if err = json.Unmarshal(serviceJson, service); err != nil {
		return
	}
	service, err = s.client.CoreV1().Services(s.namespace).Update(s.ctx, service, metav1.UpdateOptions{})
	return
}

// apply service from file
func (s *Service) Apply(filepath string) (service *corev1.Service, err error) {
	service, err = s.Create(filepath)
	if errors.IsAlreadyExists(err) {
		service, err = s.Update(filepath)
	}
	return
}

// delete service by name
func (s *Service) Delete(name string, forceDelete bool) (err error) {
	var gracePeriodSeconds int64
	if forceDelete {
		gracePeriodSeconds = 0
		err = s.client.CoreV1().Services(s.namespace).Delete(s.ctx,
			name, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds})
		return
	}
	err = s.client.CoreV1().Services(s.namespace).Delete(s.ctx,
		name, metav1.DeleteOptions{})
	return
}

// get service by name
func (s *Service) Get(name string) (service *corev1.Service, err error) {
	return
}
