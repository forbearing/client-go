package k8s

import (
	"bytes"
	"context"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func ApplyF(ctx context.Context, kubeconfig, filepath string) (err error) {
	var (
		deployment            *Deployment
		service               *Service
		pod                   *Pod
		statefulset           *StatefulSet
		daemonset             *DaemonSet
		namespace             *Namespace
		configmap             *ConfigMap
		secret                *Secret
		serviceaccount        *ServiceAccount
		ingress               *Ingress
		ingressclass          *IngressClass
		networkpolicy         *NetworkPolicy
		job                   *Job
		cronjob               *CronJob
		persistentvolume      *PersistentVolume
		persistentvolumeclaim *PersistentVolumeClaim
		clusterrole           *ClusterRole
		clusterrolebinding    *ClusterRoleBinding
		role                  *Role
		rolebinding           *RoleBinding
	)
	k8sResourceFile, err := ioutil.ReadFile(filepath)
	if err != nil {
		return
	}
	// remove all comments in the yaml file.
	removeComments := regexp.MustCompile(`#.*`)
	k8sResourceFile = removeComments.ReplaceAll(k8sResourceFile, []byte(""))
	// split yaml file by "---"
	k8sResourceItems := bytes.Split(k8sResourceFile, []byte("---"))

	for _, k8sResource := range k8sResourceItems {
		// ignore empty line
		if len(strings.TrimSpace(string(k8sResource))) == 0 {
			continue
		}
		object, err := Decode(k8sResource)
		if err != nil {
			logrus.Error("Decode error")
			logrus.Error(err)
			continue
		}
		switch object.(type) {
		case *corev1.Namespace:
			if namespace, err = NewNamespace(ctx, kubeconfig); err != nil {
				return err
			}
			if ns, err := namespace.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply namespace %q failed", ns.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply namespace %q success.", ns.Name)
			}
		case *corev1.Service:
			if service, err = NewService(ctx, "", kubeconfig); err != nil {
				return err
			}
			if svc, err := service.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply service %q failed", svc.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply service %q success.", svc.Name)
			}
		case *corev1.ConfigMap:
			if configmap, err = NewConfigMap(ctx, "", kubeconfig); err != nil {
				return err
			}
			if cm, err := configmap.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply configmap %q failed.", cm.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply configmap %q success.", cm.Name)
			}
		case *corev1.Secret:
			if secret, err = NewSecret(ctx, "", kubeconfig); err != nil {
				return err
			}
			if q, err := secret.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply secret %q failed.", q.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply secret %q success.", q.Name)
			}
		case *corev1.ServiceAccount:
			if serviceaccount, err = NewServiceAccount(ctx, "", kubeconfig); err != nil {
				return err
			}
			if sa, err := serviceaccount.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply serviceaccount %q failed", sa.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply serviceaccount %q success.", sa.Name)
			}
		case *corev1.Pod:
			if pod, err = NewPod(ctx, "", kubeconfig); err != nil {
				return err
			}
			if p, err := pod.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply pod %q failed", p.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply pod %q success.", p.Name)
			}
		case *corev1.PersistentVolume:
			if persistentvolume, err = NewPersistentVolume(ctx, kubeconfig); err != nil {
				return err
			}
			if pv, err := persistentvolume.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply persistentvolume %q failed", pv.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply persistentvolume %q success.", pv.Name)
			}
		case *corev1.PersistentVolumeClaim:
			if persistentvolumeclaim, err = NewPersistentVolumeClaim(ctx, "", kubeconfig); err != nil {
				return err
			}
			if pvc, err := persistentvolumeclaim.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply persistentvolumeclaim %q failed", pvc.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply persistentvolumeclaim %q success.", pvc.Name)
			}
		case *appsv1.Deployment:
			if deployment, err = NewDeployment(ctx, "", kubeconfig); err != nil {
				return err
			}
			if dep, err := deployment.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply deployment %q failed", dep.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply deployment %q success.", dep.Name)
			}
		case *appsv1.StatefulSet:
			if statefulset, err = NewStatefulSet(ctx, "", kubeconfig); err != nil {
				return err
			}
			if sts, err := statefulset.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply statefulset %q failed", sts.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply statefulset %q success.", sts.Name)
			}
		case *appsv1.DaemonSet:
			if daemonset, err = NewDaemonSet(ctx, "", kubeconfig); err != nil {
				return err
			}
			if ds, err := daemonset.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply daemonset %q failed", ds.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply daemonset %q success.", ds.Name)
			}
		case *networking.Ingress:
			if ingress, err = NewIngress(ctx, "", kubeconfig); err != nil {
				return err
			}
			if i, err := ingress.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply ingress %q failed", i.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply ingress %q success.", i.Name)
			}
		case *networking.IngressClass:
			if ingressclass, err = NewIngressClass(ctx, kubeconfig); err != nil {
				return err
			}
			if ic, err := ingressclass.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply ingressclass %q failed", ic.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply ingressclass %q success.", ic.Name)
			}
		case *networking.NetworkPolicy:
			if networkpolicy, err = NewNetworkPolicy(ctx, "", kubeconfig); err != nil {
				return err
			}
			if np, err := networkpolicy.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply networkpolicy %q failed", np.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply networkpolicy %q success.", np.Name)
			}
		case *batchv1.Job:
			if job, err = NewJob(ctx, "", kubeconfig); err != nil {
				return err
			}
			if j, err := job.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply job %q failed", j.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply job %q success.", j.Name)
			}
		case *batchv1.CronJob:
			if cronjob, err = NewCronJob(ctx, "", kubeconfig); err != nil {
				return err
			}
			if cj, err := cronjob.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply cronjob %q failed", cj.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply cronjob %q success.", cj.Name)
			}
		case *rbacv1.Role:
			if role, err = NewRole(ctx, "", kubeconfig); err != nil {
				return err
			}
			if r, err := role.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply role %q failed", r.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply role %q success.", r.Name)
			}
		case *rbacv1.RoleBinding:
			if rolebinding, err = NewRoleBinding(ctx, "", kubeconfig); err != nil {
				return err
			}
			if rb, err := rolebinding.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply rolebinding %q failed", rb.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply rolebinding %q success.", rb.Name)
			}
		case *rbacv1.ClusterRole:
			if clusterrole, err = NewClusterRole(ctx, kubeconfig); err != nil {
				return err
			}
			if cr, err := clusterrole.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply clusterrole %q failed", cr.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply clusterrole %q success.", cr.Name)
			}
		case *rbacv1.ClusterRoleBinding:
			if clusterrolebinding, err = NewClusterRoleBinding(ctx, kubeconfig); err != nil {
				return err
			}
			if crb, err := clusterrolebinding.ApplyFromBytes(k8sResource); err != nil {
				logrus.Errorf("apply clusterrolebinding %q failed", crb.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("apply clusterrolebinding %q success.", crb.Name)
			}
		default:
		}
	}

	return
}

func DeleteF(ctx context.Context, kubeconfig, filepath string) (err error) {
	var ( // {{{
		deployment            *Deployment
		service               *Service
		pod                   *Pod
		statefulset           *StatefulSet
		daemonset             *DaemonSet
		namespace             *Namespace
		configmap             *ConfigMap
		secret                *Secret
		serviceaccount        *ServiceAccount
		ingress               *Ingress
		ingressclass          *IngressClass
		networkpolicy         *NetworkPolicy
		job                   *Job
		cronjob               *CronJob
		persistentvolume      *PersistentVolume
		persistentvolumeclaim *PersistentVolumeClaim
		clusterrole           *ClusterRole
		clusterrolebinding    *ClusterRoleBinding
		role                  *Role
		rolebinding           *RoleBinding
	)
	k8sResourceFile, err := ioutil.ReadFile(filepath)
	if err != nil {
		return
	}
	// remove all comments in the yaml file.
	removeComments := regexp.MustCompile(`#.*`)
	k8sResourceFile = removeComments.ReplaceAll(k8sResourceFile, []byte(""))
	// split yaml file by "---"
	k8sResourceItems := bytes.Split(k8sResourceFile, []byte("---"))

	for _, k8sResource := range k8sResourceItems {
		// ignore empty line
		if len(strings.TrimSpace(string(k8sResource))) == 0 {
			continue
		}
		object, err := Decode(k8sResource)
		if err != nil {
			logrus.Error("Decode error")
			logrus.Error(err)
			continue
		}
		switch object.(type) {
		case *corev1.Namespace:
			if namespace, err = NewNamespace(ctx, kubeconfig); err != nil {
				return err
			}
			ns, _ := namespace.GetFromBytes(k8sResource)
			if err := namespace.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete namespace %q failed", ns.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete namespace %q success.", ns.Name)
			}
		case *corev1.Service:
			if service, err = NewService(ctx, "", kubeconfig); err != nil {
				return err
			}
			svc, _ := service.GetFromBytes(k8sResource)
			if err := service.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete service %q failed", svc.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete service %q success.", svc.Name)
			}
		case *corev1.ConfigMap:
			if configmap, err = NewConfigMap(ctx, "", kubeconfig); err != nil {
				return err
			}
			cm, _ := configmap.GetFromBytes(k8sResource)
			if err := configmap.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete configmap %q failed", cm.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete configmap %q success.", cm.Name)
			}
		case *corev1.Secret:
			if secret, err = NewSecret(ctx, "", kubeconfig); err != nil {
				return err
			}
			q, _ := secret.GetFromBytes(k8sResource)
			if err := secret.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete secret %q failed", q.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete secret %q success.", q.Name)
			}
		case *corev1.ServiceAccount:
			if serviceaccount, err = NewServiceAccount(ctx, "", kubeconfig); err != nil {
				return err
			}
			sa, _ := serviceaccount.GetFromBytes(k8sResource)
			if err := serviceaccount.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete serviceaccount %q failed", sa.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete serviceaccount %q success.", sa.Name)
			}
		case *corev1.Pod:
			if pod, err = NewPod(ctx, "", kubeconfig); err != nil {
				return err
			}
			p, _ := pod.GetFromBytes(k8sResource)
			if err := pod.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete pod %q failed", p.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete pod %q success.", p.Name)
			}
		case *corev1.PersistentVolume:
			if persistentvolume, err = NewPersistentVolume(ctx, kubeconfig); err != nil {
				return err
			}
			pv, _ := persistentvolume.GetFromBytes(k8sResource)
			if err := persistentvolume.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete persistentvolume %q failed", pv.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete persistentvolume %q success.", pv.Name)
			}
		case *corev1.PersistentVolumeClaim:
			if persistentvolumeclaim, err = NewPersistentVolumeClaim(ctx, "", kubeconfig); err != nil {
				return err
			}
			pvc, _ := persistentvolume.GetFromBytes(k8sResource)
			if err := persistentvolumeclaim.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete persistentvolumeclaim %q failed", pvc.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete persistentvolumeclaim %q success.", pvc.Name)
			}
		case *appsv1.Deployment:
			if deployment, err = NewDeployment(ctx, "", kubeconfig); err != nil {
				return err
			}
			deploy, _ := deployment.GetFromBytes(k8sResource)
			if err := deployment.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete deployment %q failed", deploy.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete deployment %q success.", deploy.Name)
			}
		case *appsv1.StatefulSet:
			if statefulset, err = NewStatefulSet(ctx, "", kubeconfig); err != nil {
				return err
			}
			sts, _ := statefulset.GetFromBytes(k8sResource)
			if err := statefulset.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete statefulset %q failed", sts.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete statefulset %q success.", sts.Name)
			}
		case *appsv1.DaemonSet:
			if daemonset, err = NewDaemonSet(ctx, "", kubeconfig); err != nil {
				return err
			}
			ds, _ := daemonset.GetFromBytes(k8sResource)
			if err := daemonset.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete daemonset %q failed", ds.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete daemonset %q success.", ds.Name)
			}
		case *networking.Ingress:
			if ingress, err = NewIngress(ctx, "", kubeconfig); err != nil {
				return err
			}
			i, _ := ingress.GetFromBytes(k8sResource)
			if err := ingress.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete ingress %q failed", i.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete ingress %q success.", i.Name)
			}
		case *networking.IngressClass:
			if ingressclass, err = NewIngressClass(ctx, kubeconfig); err != nil {
				return err
			}
			ic, _ := ingressclass.GetFromBytes(k8sResource)
			if err := ingressclass.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete ingressclass %q failed", ic.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete ingressclass %q success.", ic.Name)
			}
		case *networking.NetworkPolicy:
			if networkpolicy, err = NewNetworkPolicy(ctx, "", kubeconfig); err != nil {
				return err
			}
			np, _ := networkpolicy.GetFromBytes(k8sResource)
			if err := networkpolicy.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete networkpolicy %q failed", np.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete networkpolicy %q success.", np.Name)
			}
		case *batchv1.Job:
			if job, err = NewJob(ctx, "", kubeconfig); err != nil {
				return err
			}
			j, _ := job.GetFromBytes(k8sResource)
			if err := job.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete job %q failed", j.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete job %q success.", j.Name)
			}
		case *batchv1.CronJob:
			if cronjob, err = NewCronJob(ctx, "", kubeconfig); err != nil {
				return err
			}
			cj, _ := cronjob.GetFromBytes(k8sResource)
			if err := cronjob.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete cronjob %q failed", cj.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete cronjob %q success.", cj.Name)
			}
		case *rbacv1.Role:
			if role, err = NewRole(ctx, "", kubeconfig); err != nil {
				return err
			}
			r, _ := role.GetFromBytes(k8sResource)
			if err := role.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete role %q failed", r.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete role %q success.", r.Name)
			}
		case *rbacv1.RoleBinding:
			if rolebinding, err = NewRoleBinding(ctx, "", kubeconfig); err != nil {
				return err
			}
			rb, _ := rolebinding.CreateFromBytes(k8sResource)
			if err := rolebinding.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete rolebinding %q failed", rb.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete rolebinding %q success.", rb.Name)
			}
		case *rbacv1.ClusterRole:
			if clusterrole, err = NewClusterRole(ctx, kubeconfig); err != nil {
				return err
			}
			cr, _ := clusterrole.GetFromBytes(k8sResource)
			if err := clusterrole.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete clusterrole %q failed", cr.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete clusterrole %q success.", cr.Name)
			}
		case *rbacv1.ClusterRoleBinding:
			if clusterrolebinding, err = NewClusterRoleBinding(ctx, kubeconfig); err != nil {
				return err
			}
			crb, _ := clusterrolebinding.GetFromBytes(k8sResource)
			if err := clusterrolebinding.DeleteFromBytes(k8sResource); err != nil {
				logrus.Errorf("delete clusterrolebinding %q failed", crb.Name)
				logrus.Error(err)
			} else {
				logrus.Tracef("delete clusterrolebinding %q success.", crb.Name)
			}
		default:
		}
	}

	return
} // }}}

func Decode(data []byte) (object runtime.Object, err error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	object, _, err = decode(data, nil, nil)
	return
}
