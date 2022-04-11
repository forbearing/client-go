package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	apply_appsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	apply_metav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

//type DeploymentConditionApplyConfiguration struct {
//    Type               *v1.DeploymentConditionType `json:"type,omitempty"`
//    Status             *corev1.ConditionStatus     `json:"status,omitempty"`
//    LastUpdateTime     *metav1.Time                `json:"lastUpdateTime,omitempty"`
//    LastTransitionTime *metav1.Time                `json:"lastTransitionTime,omitempty"`
//    Reason             *string                     `json:"reason,omitempty"`
//    Message            *string                     `json:"message,omitempty"`
//}
// extract the condition subresource from appsv1.Deployment
func ExtractDeploymentCondition(deployment *appsv1.Deployment) []apply_appsv1.DeploymentConditionApplyConfiguration {
	var conditionApplyList []apply_appsv1.DeploymentConditionApplyConfiguration
	deploymentConditions := deployment.Status.Conditions
	for _, condition := range deploymentConditions {
		conditionApply := apply_appsv1.DeploymentConditionApplyConfiguration{
			Type:               &condition.Type,
			Status:             &condition.Status,
			LastUpdateTime:     &condition.LastTransitionTime,
			LastTransitionTime: &condition.LastTransitionTime,
			Reason:             &condition.Reason,
			Message:            &condition.Message,
		}
		conditionApplyList = append(conditionApplyList, conditionApply)
	}
	return conditionApplyList
}

//type DeploymentStrategyApplyConfiguration struct {
//    Type          *v1.DeploymentStrategyType                 `json:"type,omitempty"`
//    RollingUpdate *RollingUpdateDeploymentApplyConfiguration `json:"rollingUpdate,omitempty"`
//}
//type RollingUpdateDeploymentApplyConfiguration struct {
//    MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
//    MaxSurge       *intstr.IntOrString `json:"maxSurge,omitempty"`
//}
// extract the strategy subresource from appsv1.Deployment
func ExtractDeploymentStrategy(deployment *appsv1.Deployment) *apply_appsv1.DeploymentStrategyApplyConfiguration {
	strategy := deployment.Spec.Strategy
	return &apply_appsv1.DeploymentStrategyApplyConfiguration{
		Type: &strategy.Type,
		RollingUpdate: &apply_appsv1.RollingUpdateDeploymentApplyConfiguration{
			MaxUnavailable: strategy.RollingUpdate.MaxSurge,
			MaxSurge:       strategy.RollingUpdate.MaxSurge,
		},
	}
}

//type LabelSelectorApplyConfiguration struct {
//    MatchLabels      map[string]string                            `json:"matchLabels,omitempty"`
//    MatchExpressions []LabelSelectorRequirementApplyConfiguration `json:"matchExpressions,omitempty"`
//}
//type LabelSelectorRequirementApplyConfiguration struct {
//    Key      *string                   `json:"key,omitempty"`
//    Operator *v1.LabelSelectorOperator `json:"operator,omitempty"`
//    Values   []string                  `json:"values,omitempty"`
//}
// extract the labelSelector subresource from appsv1.Deployment
func ExtractDeploymentLabelSelector(deployment *appsv1.Deployment) *apply_metav1.LabelSelectorApplyConfiguration {
	selector := deployment.Spec.Selector
	var matchExpressions = []apply_metav1.LabelSelectorRequirementApplyConfiguration{}
	for _, expression := range selector.MatchExpressions {
		expressApply := apply_metav1.LabelSelectorRequirementApplyConfiguration{
			Key:      &expression.Key,
			Operator: &expression.Operator,
			Values:   expression.Values,
		}
		matchExpressions = append(matchExpressions, expressApply)
	}
	return &apply_metav1.LabelSelectorApplyConfiguration{
		MatchLabels:      selector.MatchLabels,
		MatchExpressions: matchExpressions,
	}
}
