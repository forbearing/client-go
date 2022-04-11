package k8s

//func f1() {
//    //type DeploymentStatusApplyConfiguration struct {
//    //    ObservedGeneration  *int64                                  `json:"observedGeneration,omitempty"`
//    //    Replicas            *int32                                  `json:"replicas,omitempty"`
//    //    UpdatedReplicas     *int32                                  `json:"updatedReplicas,omitempty"`
//    //    ReadyReplicas       *int32                                  `json:"readyReplicas,omitempty"`
//    //    AvailableReplicas   *int32                                  `json:"availableReplicas,omitempty"`
//    //    UnavailableReplicas *int32                                  `json:"unavailableReplicas,omitempty"`
//    //    Conditions          []DeploymentConditionApplyConfiguration `json:"conditions,omitempty"`
//    //    CollisionCount      *int32                                  `json:"collisionCount,omitempty"`
//    //}
//    deploymentStatusApplyConfiguration := &apply_appsv1.DeploymentStatusApplyConfiguration{
//        ObservedGeneration:  &deploy.Status.ObservedGeneration,
//        Replicas:            &deploy.Status.Replicas,
//        UpdatedReplicas:     &deploy.Status.UpdatedReplicas,
//        ReadyReplicas:       &deploy.Status.ReadyReplicas,
//        AvailableReplicas:   &deploy.Status.AvailableReplicas,
//        UnavailableReplicas: &deploy.Status.UnavailableReplicas,
//        Conditions:          extractapply_appsv1.ExtractDeploymentCondition(deploy),
//        CollisionCount:      deploy.Status.CollisionCount,
//    }

//    //type DeploymentSpecApplyConfiguration struct {
//    //    Replicas                *int32                                    `json:"replicas,omitempty"`
//    //    Selector                *v1.LabelSelectorApplyConfiguration       `json:"selector,omitempty"`
//    //    Template                *corev1.PodTemplateSpecApplyConfiguration `json:"template,omitempty"`
//    //    Strategy                *DeploymentStrategyApplyConfiguration     `json:"strategy,omitempty"`
//    //    MinReadySeconds         *int32                                    `json:"minReadySeconds,omitempty"`
//    //    RevisionHistoryLimit    *int32                                    `json:"revisionHistoryLimit,omitempty"`
//    //    Paused                  *bool                                     `json:"paused,omitempty"`
//    //    ProgressDeadlineSeconds *int32                                    `json:"progressDeadlineSeconds,omitempty"`
//    //}
//    deploymentSpecApplyConfiguration := &apply_appsv1.DeploymentSpecApplyConfiguration{
//        Replicas:                deploy.Spec.Replicas,
//        Selector:                extractapply_appsv1.ExtractDeploymentLabelSelector(deploy),
//        Template:                nil,
//        Strategy:                extractapply_appsv1.ExtractDeploymentStrategy(deploy),
//        MinReadySeconds:         &deploy.Spec.MinReadySeconds,
//        RevisionHistoryLimit:    deploy.Spec.RevisionHistoryLimit,
//        Paused:                  &deploy.Spec.Paused,
//        ProgressDeadlineSeconds: deploy.Spec.ProgressDeadlineSeconds,
//    }

//    //type OwnerReferenceApplyConfiguration struct {
//    //    APIVersion         *string    `json:"apiVersion,omitempty"`
//    //    Kind               *string    `json:"kind,omitempty"`
//    //    Name               *string    `json:"name,omitempty"`
//    //    UID                *types.UID `json:"uid,omitempty"`
//    //    Controller         *bool      `json:"controller,omitempty"`
//    //    BlockOwnerDeletion *bool      `json:"blockOwnerDeletion,omitempty"`
//    //}
//    //type TypeMetaApplyConfiguration struct {
//    //    Kind       *string `json:"kind,omitempty"`
//    //    APIVersion *string `json:"apiVersion,omitempty"`
//    //}
//    typeMetaApplyConfiguration := apply_metav1.TypeMetaApplyConfiguration{
//        Kind:       &deploy.Kind,
//        APIVersion: &deploy.APIVersion,
//    }

//    //type ObjectMetaApplyConfiguration struct {
//    //    Name                       *string                            `json:"name,omitempty"`
//    //    GenerateName               *string                            `json:"generateName,omitempty"`
//    //    Namespace                  *string                            `json:"namespace,omitempty"`
//    //    SelfLink                   *string                            `json:"selfLink,omitempty"`
//    //    UID                        *types.UID                         `json:"uid,omitempty"`
//    //    ResourceVersion            *string                            `json:"resourceVersion,omitempty"`
//    //    Generation                 *int64                             `json:"generation,omitempty"`
//    //    CreationTimestamp          *v1.Time                           `json:"creationTimestamp,omitempty"`
//    //    DeletionTimestamp          *v1.Time                           `json:"deletionTimestamp,omitempty"`
//    //    DeletionGracePeriodSeconds *int64                             `json:"deletionGracePeriodSeconds,omitempty"`
//    //    Labels                     map[string]string                  `json:"labels,omitempty"`
//    //    Annotations                map[string]string                  `json:"annotations,omitempty"`
//    //    OwnerReferences            []OwnerReferenceApplyConfiguration `json:"ownerReferences,omitempty"`
//    //    Finalizers                 []string                           `json:"finalizers,omitempty"`
//    //    ClusterName                *string                            `json:"clusterName,omitempty"`
//    //}
//    objectMetaApplyConfiguration := &apply_metav1.ObjectMetaApplyConfiguration{
//        Name:                       &deploy.Name,
//        GenerateName:               &deploy.GenerateName,
//        Namespace:                  &deploy.Namespace,
//        SelfLink:                   &deploy.SelfLink,
//        UID:                        &deploy.UID,
//        ResourceVersion:            &deploy.ResourceVersion,
//        Generation:                 &deploy.Generation,
//        CreationTimestamp:          &deploy.CreationTimestamp,
//        DeletionTimestamp:          deploy.DeletionTimestamp,
//        DeletionGracePeriodSeconds: deploy.DeletionGracePeriodSeconds,
//        Labels:                     deploy.Labels,
//        Annotations:                deploy.Annotations,
//        // k8s controllers(deployment,statefulset, cronjob, etc.) doesn't have metadata.ownerReferences field.
//        // dependent objects(pods) have a metadata.ownerReferences field.
//        // more information about "metadata.ownerReferences" refer to bellow link
//        // https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
//        OwnerReferences: nil,
//        Finalizers:      deploy.Finalizers,
//        ClusterName:     &deploy.ClusterName,
//    }

//    //type DeploymentApplyConfiguration struct {
//    //    v1.TypeMetaApplyConfiguration    `json:",inline"`
//    //    *v1.ObjectMetaApplyConfiguration `json:"metadata,omitempty"`
//    //    Spec                             *DeploymentSpecApplyConfiguration   `json:"spec,omitempty"`
//    //    Status                           *DeploymentStatusApplyConfiguration `json:"status,omitempty"`
//    //}
//    var deploymentApplyConfiguration *apply_appsv1.DeploymentApplyConfiguration
//    deploymentApplyConfiguration = &apply_appsv1.DeploymentApplyConfiguration{
//        TypeMetaApplyConfiguration:   typeMetaApplyConfiguration,
//        ObjectMetaApplyConfiguration: objectMetaApplyConfiguration,
//        Spec:                         deploymentSpecApplyConfiguration,
//        Status:                       deploymentStatusApplyConfiguration,
//    }
//}
