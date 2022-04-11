package k8s

// references:
//	https://stackoverflow.com/questions/40975307/how-to-watch-events-on-a-kubernetes-service-using-its-go-client
//	https://github.com/kubernetes/client-go/issues/132
//	https://github.com/kubernetes/client-go/issues/623

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// corev1.NamespaceDefault
// corev1.NamespaceAll
// corev1.NamespaceActive

//const (
//    // NamespaceDefault means the object is in the default namespace which is applied when not specified by clients
//    NamespaceDefault string = "default"
//    // NamespaceAll is the default argument to specify on a context when you want to list or filter resources across all namespaces
//    NamespaceAll string = ""
//    // NamespaceNodeLease is the namespace where we place node lease objects (used for node heartbeats)
//    NamespaceNodeLease string = "kube-node-lease"
//)

//// These are the valid phases of a namespace.
//const (
//    // NamespaceActive means the namespace is available for use in the system
//    NamespaceActive NamespacePhase = "Active"
//    // NamespaceTerminating means the namespace is undergoing graceful termination
//    NamespaceTerminating NamespacePhase = "Terminating"
//)

func DeploymentInformer(ctx context.Context, clientset *kubernetes.Clientset, namespace string) {
	watchList := cache.NewListWatchFromClient(
		clientset.AppsV1().RESTClient(),
		"deployments",
		namespace,
		fields.Everything(),
	)
	_, controller := cache.NewInformer(
		watchList,
		&appsv1.Deployment{},
		0, // Duration is int64
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				myObj := obj.(metav1.Object)
				logrus.Infof("deployment add: %s\n", myObj.GetName())
			},
			DeleteFunc: func(obj interface{}) {
				myObj := obj.(metav1.Object)
				fmt.Printf("Deployment deleted: %s\n", myObj.GetName())
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				oObj := oldObj.(metav1.Object)
				nObj := newObj.(metav1.Object)
				logrus.Infof("Deployment changed, %s to %s\n", oObj.GetName(), nObj.GetName())
			},
		},
	)
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(stop)
	for {
		time.Sleep(time.Second)
	}

}

func ServiceInformer(ctx context.Context, clientset *kubernetes.Clientset, namespace string) {
	watchList := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		string(corev1.ResourceServices),
		namespace,
		fields.Everything(),
	)

	_, controller := cache.NewInformer( // also take a look at NewSharedIndexInformer
		watchList,
		&corev1.Service{},
		0, //Duration is int64
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				myObj := obj.(metav1.Object)
				logrus.Infof("service added: %s\n", myObj.GetName())
			},
			DeleteFunc: func(obj interface{}) {
				myObj := obj.(metav1.Object)
				logrus.Infof("service deleted: %s \n", myObj.GetName())
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				oObj := oldObj.(metav1.Object)
				nObj := newObj.(metav1.Object)
				logrus.Infof("service changed, %s to %s \n", oObj.GetName(), nObj.GetName())
			},
		},
	)
	// I found it in k8s scheduler module. Maybe it's help if you interested in.
	// serviceInformer := cache.NewSharedIndexInformer(watchlist, &v1.Service{}, 0, cache.Indexers{
	//     cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	// })
	// go serviceInformer.Run(stop)
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(stop)
	for {
		time.Sleep(time.Second)
	}
}
