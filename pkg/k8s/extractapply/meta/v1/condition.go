package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	_ "k8s.io/client-go/applyconfigurations/meta/v1"
	metav1_apply "k8s.io/client-go/applyconfigurations/meta/v1"
)

func ExtraConditions(conditions interface{}) metav1_apply.ConditionApplyConfiguration {
	//deployConditions, ok := conditions.([]appsv1.DeploymentCondition)
	//if ok {
	//    return extraDeploymentConditions(deployConditions)
	//}
	return metav1_apply.ConditionApplyConfiguration{}
}
func extraDeploymentConditions(conditions []appsv1.DeploymentCondition) metav1_apply.ConditionApplyConfiguration {
	return metav1_apply.ConditionApplyConfiguration{}
}
