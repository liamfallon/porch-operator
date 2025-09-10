/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"os"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "github.com/liamfallon/porch-operator/api/v1alpha1"
)

const PackageRevisionFinalizer = "cache.example.com/finalizer"

// Definitions to manage status conditions
const (
	// typeAvailablePackageRevision represents the status of the Deployment reconciliation
	typeAvailablePackageRevision = "Available"
	// typeDegradedPackageRevision represents the status used when the custom resource is deleted and the finalizer operations are yet to occur.
	typeDegradedPackageRevision = "Degraded"
)

// PackageRevisionReconciler reconciles a PackageRevision object
type PackageRevisionReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// The following markers are used to generate the rules permissions (RBAC) on config/rbac using controller-gen
// when the command <make manifests> is executed.
// To know more about markers see: https://book.kubebuilder.io/reference/markers.html

// +kubebuilder:rbac:groups=cache.example.com,resources=PackageRevisions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=PackageRevisions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.example.com,resources=PackageRevisions/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// It is essential for the controller's reconciliation loop to be idempotent. By following the Operator
// pattern you will create Controllers which provide a reconcile function
// responsible for synchronizing resources until the desired state is reached on the cluster.
// Breaking this recommendation goes against the design principles of controller-runtime.
// and may lead to unforeseen consequences such as resources becoming stuck and requiring manual intervention.
// For further info:
// - About Operator Pattern: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
// - About Controllers: https://kubernetes.io/docs/concepts/architecture/controller/
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *PackageRevisionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the PackageRevision instance
	// The purpose is check if the Custom Resource for the Kind PackageRevision
	// is applied on the cluster if not we return nil to stop the reconciliation
	PackageRevision := &cachev1alpha1.PackageRevision{}
	err := r.Get(ctx, req.NamespacedName, PackageRevision)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("PackageRevision resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get PackageRevision")
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status is available
	if len(PackageRevision.Status.Conditions) == 0 {
		meta.SetStatusCondition(&PackageRevision.Status.Conditions, metav1.Condition{Type: typeAvailablePackageRevision, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err = r.Status().Update(ctx, PackageRevision); err != nil {
			log.Error(err, "Failed to update PackageRevision status")
			return ctrl.Result{}, err
		}

		// Let's re-fetch the PackageRevision Custom Resource after updating the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raising the error "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.Get(ctx, req.NamespacedName, PackageRevision); err != nil {
			log.Error(err, "Failed to re-fetch PackageRevision")
			return ctrl.Result{}, err
		}
	}

	// Let's add a finalizer. Then, we can define some operations which should
	// occur before the custom resource is deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if !controllerutil.ContainsFinalizer(PackageRevision, PackageRevisionFinalizer) {
		log.Info("Adding Finalizer for PackageRevision")
		if ok := controllerutil.AddFinalizer(PackageRevision, PackageRevisionFinalizer); !ok {
			err = fmt.Errorf("finalizer for PackageRevision was not added")
			log.Error(err, "Failed to add finalizer for PackageRevision")
			return ctrl.Result{}, err
		}

		if err = r.Update(ctx, PackageRevision); err != nil {
			log.Error(err, "Failed to update custom resource to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Check if the PackageRevision instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isPackageRevisionMarkedToBeDeleted := PackageRevision.GetDeletionTimestamp() != nil
	if isPackageRevisionMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(PackageRevision, PackageRevisionFinalizer) {
			log.Info("Performing Finalizer Operations for PackageRevision before delete CR")

			// Let's add here a status "Downgrade" to reflect that this resource began its process to be terminated.
			meta.SetStatusCondition(&PackageRevision.Status.Conditions, metav1.Condition{Type: typeDegradedPackageRevision,
				Status: metav1.ConditionUnknown, Reason: "Finalizing",
				Message: fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", PackageRevision.Name)})

			if err := r.Status().Update(ctx, PackageRevision); err != nil {
				log.Error(err, "Failed to update PackageRevision status")
				return ctrl.Result{}, err
			}

			// Perform all operations required before removing the finalizer and allow
			// the Kubernetes API to remove the custom resource.
			r.doFinalizerOperationsForPackageRevision(PackageRevision)

			// TODO(user): If you add operations to the doFinalizerOperationsForPackageRevision method
			// then you need to ensure that all worked fine before deleting and updating the Downgrade status
			// otherwise, you should requeue here.

			// Re-fetch the PackageRevision Custom Resource before updating the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raising the error "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, PackageRevision); err != nil {
				log.Error(err, "Failed to re-fetch PackageRevision")
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&PackageRevision.Status.Conditions, metav1.Condition{Type: typeDegradedPackageRevision,
				Status: metav1.ConditionTrue, Reason: "Finalizing",
				Message: fmt.Sprintf("Finalizer operations for custom resource %s name were successfully accomplished", PackageRevision.Name)})

			if err := r.Status().Update(ctx, PackageRevision); err != nil {
				log.Error(err, "Failed to update PackageRevision status")
				return ctrl.Result{}, err
			}

			log.Info("Removing Finalizer for PackageRevision after successfully perform the operations")
			if ok := controllerutil.RemoveFinalizer(PackageRevision, PackageRevisionFinalizer); !ok {
				err = fmt.Errorf("finalizer for PackageRevision was not removed")
				log.Error(err, "Failed to remove finalizer for PackageRevision")
				return ctrl.Result{}, err
			}

			if err := r.Update(ctx, PackageRevision); err != nil {
				log.Error(err, "Failed to remove finalizer for PackageRevision")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// The CRD API defines that the PackageRevision type have a PackageRevisionSpec.Size field
	// to set the quantity of Deployment instances to the desired state on the cluster.
	// Therefore, the following code will ensure the Deployment size is the same as defined
	// via the Size spec of the Custom Resource which we are reconciling.
	lifecycle := PackageRevision.Spec.Lifecycle
	klog.Infof("Lifecycle is %s", lifecycle)

	// The following implementation will update the status
	meta.SetStatusCondition(&PackageRevision.Status.Conditions, metav1.Condition{Type: typeAvailablePackageRevision,
		Status: metav1.ConditionTrue, Reason: "Reconciling",
		Message: fmt.Sprintf("Deployment for custom resource (%s) with %s lifecycle successful", PackageRevision.Name, lifecycle)})

	if err := r.Status().Update(ctx, PackageRevision); err != nil {
		log.Error(err, "Failed to update PackageRevision status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// finalizePackageRevision will perform the required operations before delete the CR.
func (r *PackageRevisionReconciler) doFinalizerOperationsForPackageRevision(cr *cachev1alpha1.PackageRevision) {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.

	// Note: It is not recommended to use finalizers with the purpose of deleting resources which are
	// created and managed in the reconciliation. These ones, such as the Deployment created on this reconcile,
	// are defined as dependent of the custom resource. See that we use the method ctrl.SetControllerReference.
	// to set the ownerRef which means that the Deployment will be deleted by the Kubernetes API.
	// More info: https://kubernetes.io/docs/tasks/administer-cluster/use-cascading-deletion/

	// The following implementation will raise an event
	r.Recorder.Event(cr, "Warning", "Deleting",
		fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s",
			cr.Name,
			cr.Namespace))
}

// labelsForPackageRevision returns the labels for selecting the resources
// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
func labelsForPackageRevision() map[string]string {
	var imageTag string
	image, err := imageForPackageRevision()
	if err == nil {
		imageTag = strings.Split(image, ":")[1]
	}
	return map[string]string{
		"app.kubernetes.io/name":       "PackageRevision-operator",
		"app.kubernetes.io/version":    imageTag,
		"app.kubernetes.io/managed-by": "PackageRevisionController",
	}
}

// imageForPackageRevision gets the Operand image which is managed by this controller
// from the PackageRevision_IMAGE environment variable defined in the config/manager/manager.yaml
func imageForPackageRevision() (string, error) {
	var imageEnvVar = "PackageRevision_IMAGE"
	image, found := os.LookupEnv(imageEnvVar)
	if !found {
		return "", fmt.Errorf("unable to find %s environment variable with the image", imageEnvVar)
	}
	return image, nil
}

// SetupWithManager sets up the controller with the Manager.
// The whole idea is to be watching the resources that matter for the controller.
// When a resource that the controller is interested in changes, the Watch triggers
// the controller’s reconciliation loop, ensuring that the actual state of the resource
// matches the desired state as defined in the controller’s logic.
//
// Notice how we configured the Manager to monitor events such as the creation, update,
// or deletion of a Custom Resource (CR) of the PackageRevision kind, as well as any changes
// to the Deployment that the controller manages and owns.
func (r *PackageRevisionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Watch the PackageRevision CR(s) and trigger reconciliation whenever it
		// is created, updated, or deleted
		For(&cachev1alpha1.PackageRevision{}).
		Named("PackageRevision").
		// Watch the Deployment managed by the PackageRevisionReconciler. If any changes occur to the Deployment
		// owned and managed by this controller, it will trigger reconciliation, ensuring that the cluster
		// state aligns with the desired state. See that the ownerRef was set when the Deployment was created.
		Owns(&appsv1.Deployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
