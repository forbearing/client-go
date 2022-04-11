package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/applyconfigurations/core/v1"
	apply_corev1 "k8s.io/client-go/applyconfigurations/core/v1"
	apply_metav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

//type PodTemplateSpecApplyConfiguration struct {
//    *v1.ObjectMetaApplyConfiguration `json:"metadata,omitempty"`
//    Spec                             *PodSpecApplyConfiguration `json:"spec,omitempty"`
//}
//type ObjectMetaApplyConfiguration struct {
//    Name                       *string                            `json:"name,omitempty"`
//    GenerateName               *string                            `json:"generateName,omitempty"`
//    Namespace                  *string                            `json:"namespace,omitempty"`
//    SelfLink                   *string                            `json:"selfLink,omitempty"`
//    UID                        *types.UID                         `json:"uid,omitempty"`
//    ResourceVersion            *string                            `json:"resourceVersion,omitempty"`
//    Generation                 *int64                             `json:"generation,omitempty"`
//    CreationTimestamp          *v1.Time                           `json:"creationTimestamp,omitempty"`
//    DeletionTimestamp          *v1.Time                           `json:"deletionTimestamp,omitempty"`
//    DeletionGracePeriodSeconds *int64                             `json:"deletionGracePeriodSeconds,omitempty"`
//    Labels                     map[string]string                  `json:"labels,omitempty"`
//    Annotations                map[string]string                  `json:"annotations,omitempty"`
//    OwnerReferences            []OwnerReferenceApplyConfiguration `json:"ownerReferences,omitempty"`
//    Finalizers                 []string                           `json:"finalizers,omitempty"`
//    ClusterName                *string                            `json:"clusterName,omitempty"`
//}
//type OwnerReferenceApplyConfiguration struct {
//    APIVersion         *string    `json:"apiVersion,omitempty"`
//    Kind               *string    `json:"kind,omitempty"`
//    Name               *string    `json:"name,omitempty"`
//    UID                *types.UID `json:"uid,omitempty"`
//    Controller         *bool      `json:"controller,omitempty"`
//    BlockOwnerDeletion *bool      `json:"blockOwnerDeletion,omitempty"`
//}
// extract pod template subresource from k8s controllers(deployment, statefulset, daemonset, job, cronjob)
func ExtractPodTemplate(object interface{}) *apply_corev1.PodTemplateApplyConfiguration {
	var spec corev1.PodTemplateSpec
	switch obj := object.(type) {
	case *appsv1.Deployment:
		spec = obj.Spec.Template
	case *appsv1.StatefulSet:
		spec = obj.Spec.Template
	case *appsv1.DaemonSet:
		spec = obj.Spec.Template
	case *appsv1.ReplicaSet:
		spec = obj.Spec.Template
	case *batchv1.Job:
		spec = obj.Spec.Template
	case *batchv1.CronJob:
		//spec = obj.Spec.Template
	}
	var ownerReferences []apply_metav1.OwnerReferenceApplyConfiguration
	for _, or := range spec.OwnerReferences {
		orApply := apply_metav1.OwnerReferenceApplyConfiguration{
			APIVersion:         &or.APIVersion,
			Kind:               &or.Kind,
			Name:               &or.Name,
			UID:                &or.UID,
			Controller:         or.Controller,
			BlockOwnerDeletion: or.BlockOwnerDeletion,
		}
		ownerReferences = append(ownerReferences, orApply)
	}
	podTemplateApplyConfiguration := &apply_corev1.PodTemplateApplyConfiguration{}
	podTemplateApplyConfiguration.ObjectMetaApplyConfiguration = &apply_metav1.ObjectMetaApplyConfiguration{
		Name:                       &spec.Name,
		GenerateName:               &spec.GenerateName,
		Namespace:                  &spec.Namespace,
		SelfLink:                   &spec.SelfLink,
		UID:                        &spec.UID,
		ResourceVersion:            &spec.ResourceVersion,
		Generation:                 &spec.Generation,
		CreationTimestamp:          &spec.CreationTimestamp,
		DeletionTimestamp:          spec.DeletionTimestamp,
		DeletionGracePeriodSeconds: spec.DeletionGracePeriodSeconds,
		Labels:                     spec.Labels,
		Annotations:                spec.Annotations,
		OwnerReferences:            ownerReferences,
		Finalizers:                 spec.Finalizers,
		ClusterName:                &spec.ClusterName,
	}
	return podTemplateApplyConfiguration
}
