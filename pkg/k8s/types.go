package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HandlerInterface interface {
	CreateFromBytes(data []byte) (*appsv1.Deployment, error)
	CreateFromFile(path string) (*appsv1.Deployment, error)
	Create(path string) (*appsv1.Deployment, error)

	UpdateFromBytes(data []byte) (*appsv1.Deployment, error)
	UpdateFromFile(path string) (appsv1.Deployment, error)
	Update(path string) (*appsv1.Deployment, error)

	DeleteByName(data []byte) error
	DeleteFromBytes(data []byte) error
	DeleteFromFile(path string) error
	Delete(name string) error

	GetByName(name string) (*appsv1.Deployment, error)
	GetFromBytes(name string) (*appsv1.Deployment, error)
	GetFromFile(path string) (*appsv1.Deployment, error)
	Get(name string) (*appsv1.Deployment, error)

	List(label string) (*appsv1.DeploymentList, error)

	WatchByName(name string, addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) error
	WatchByLabel(label string, addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) error
	Watch(name string, addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) error
}

// k8s resource name
const (
	// controller
	ResourceKindPod                   = "pod"
	ResourceKindDeployment            = "deployment"
	ResourceKindDaemonSet             = "daemonset"
	ResourceKindStatefulSet           = "statefulset"
	ResourceKindJob                   = "job"
	ResourceKindCronJob               = "cronjob"
	ResourceKindReplicaSet            = "replicaset"
	ResourceKindReplicationController = "replicationcontroller"

	ResourceKindPods                   = "pods"
	ResourceKindDeployments            = "deployments"
	ResourceKindDaemonSets             = "daemonsets"
	ResourceKindStatefulSets           = "statefulsets"
	ResourceKindJobs                   = "jobs"
	ResourceKindCronJobs               = "cronjobs"
	ResourceKindReplicaSets            = "replicasets"
	ResourceKindReplicationControllers = "replicationcontrollers"

	//CLUSTERROLEBINDING    = "clusterrolebinding"
	//CLUSTERROLE           = "clusterrole"
	//CONFIGMAP             = "configmap"
	//INGRES                = "ingress"
	//INGRESSCLASS          = "ingressclass"
	//NAMESPACE             = "namespace"
	//NETWORKPOLICY         = "networkpolicy"
	//NODE                  = "node"
	//PERSISTENTVOLUMECLAIM = "persistentvolumeclaim"
	//PERSISTENTVOLUME      = "persistentvolume"
	//ROLEBINDING           = "rolebinding"
	//ROLE                  = "role"
	//SECRET                = "secret"
	//SERVICEACCOUNT        = "serviceaccount"
	//SERVICE               = "service"
)

//type ListOptions struct {
//    TypeMeta             `json:",inline"`
//    LabelSelector        string               `json:"labelSelector,omitempty" protobuf:"bytes,1,opt,name=labelSelector"`
//    FieldSelector        string               `json:"fieldSelector,omitempty" protobuf:"bytes,2,opt,name=fieldSelector"`
//    Watch                bool                 `json:"watch,omitempty" protobuf:"varint,3,opt,name=watch"`
//    AllowWatchBookmarks  bool                 `json:"allowWatchBookmarks,omitempty" protobuf:"varint,9,opt,name=allowWatchBookmarks"`
//    ResourceVersion      string               `json:"resourceVersion,omitempty" protobuf:"bytes,4,opt,name=resourceVersion"`
//    ResourceVersionMatch ResourceVersionMatch `json:"resourceVersionMatch,omitempty" protobuf:"bytes,10,opt,name=resourceVersionMatch,casttype=ResourceVersionMatch"`
//    TimeoutSeconds       *int64               `json:"timeoutSeconds,omitempty" protobuf:"varint,5,opt,name=timeoutSeconds"`
//    Limit                int64                `json:"limit,omitempty" protobuf:"varint,7,opt,name=limit"`
//    Continue             string               `json:"continue,omitempty" protobuf:"bytes,8,opt,name=continue"`
//}
//type GetOptions struct {
//    TypeMeta        `json:",inline"`
//    ResourceVersion string `json:"resourceVersion,omitempty" protobuf:"bytes,1,opt,name=resourceVersion"`
//}
//type DeleteOptions struct {
//    TypeMeta           `json:",inline"`
//    GracePeriodSeconds *int64               `json:"gracePeriodSeconds,omitempty" protobuf:"varint,1,opt,name=gracePeriodSeconds"`
//    Preconditions      *Preconditions       `json:"preconditions,omitempty" protobuf:"bytes,2,opt,name=preconditions"`
//    OrphanDependents   *bool                `json:"orphanDependents,omitempty" protobuf:"varint,3,opt,name=orphanDependents"`
//    PropagationPolicy  *DeletionPropagation `json:"propagationPolicy,omitempty" protobuf:"varint,4,opt,name=propagationPolicy"`
//    DryRun             []string             `json:"dryRun,omitempty" protobuf:"bytes,5,rep,name=dryRun"`
//}
//type CreateOptions struct {
//    TypeMeta        `json:",inline"`
//    DryRun          []string `json:"dryRun,omitempty" protobuf:"bytes,1,rep,name=dryRun"`
//    FieldManager    string   `json:"fieldManager,omitempty" protobuf:"bytes,3,name=fieldManager"`
//    FieldValidation string   `json:"fieldValidation,omitempty" protobuf:"bytes,4,name=fieldValidation"`
//}
//type PatchOptions struct {
//    TypeMeta        `json:",inline"`
//    DryRun          []string `json:"dryRun,omitempty" protobuf:"bytes,1,rep,name=dryRun"`
//    Force           *bool    `json:"force,omitempty" protobuf:"varint,2,opt,name=force"`
//    FieldManager    string   `json:"fieldManager,omitempty" protobuf:"bytes,3,name=fieldManager"`
//    FieldValidation string   `json:"fieldValidation,omitempty" protobuf:"bytes,4,name=fieldValidation"`
//}
//type ApplyOptions struct {
//    TypeMeta     `json:",inline"`
//    DryRun       []string `json:"dryRun,omitempty" protobuf:"bytes,1,rep,name=dryRun"`
//    Force        bool     `json:"force" protobuf:"varint,2,opt,name=force"`
//    FieldManager string   `json:"fieldManager" protobuf:"bytes,3,name=fieldManager"`
//}
//type UpdateOptions struct {
//    TypeMeta        `json:",inline"`
//    DryRun          []string `json:"dryRun,omitempty" protobuf:"bytes,1,rep,name=dryRun"`
//    FieldManager    string   `json:"fieldManager,omitempty" protobuf:"bytes,2,name=fieldManager"`
//    FieldValidation string   `json:"fieldValidation,omitempty" protobuf:"bytes,3,name=fieldValidation"`
//}

type HandlerOptions struct {
	ListOptions   metav1.ListOptions
	GetOptions    metav1.GetOptions
	CreateOptions metav1.CreateOptions
	DeleteOptions metav1.DeleteOptions
	ApplyOptions  metav1.ApplyOptions
	UpdateOptions metav1.UpdateOptions
	PatchOptions  metav1.PatchOptions
}

const (
	FieldManager = "client-go"
)

const (
	// LabelNodeRolePrefix is a label prefix for node roles
	// It's copied over to here until it's merged in core: https://github.com/kubernetes/kubernetes/pull/39112
	LabelNodeRolePrefix = "node-role.kubernetes.io/"

	// LabelNodeRole specifies the role of a node
	LabelNodeRole = "kubernetes.io/role"

	//LabelNodeRoleMaster       = "kubernetes.io/role=master"
	//LabelNodeRoleControlPlane = "kubernetes.io/role=control-plane"
	//LabelNodeRoleWorker       = "!kubernetes.io/role=master"

	NodeRoleMaster       = "master"
	NodeRoleControlPlane = "control-plane"
)

type NodeStatus struct {
	Status  string
	Message string
	Reason  string
}
type NodeInfo struct {
	Hostname           string
	IPAddress          string
	AllocatableCpu     string
	AllocatableMemory  string
	AllocatableStorage string
	TotalCpu           string
	TotalMemory        string
	TotalStorage       string

	Architecture            string
	BootID                  string
	ContainerRuntimeVersion string
	KernelVersion           string
	KubeProxyVersion        string
	KubeletVersion          string
	MachineID               string
	OperatingSystem         string
	OSImage                 string
	SystemUUID              string
}
