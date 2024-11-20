/*
Copyright 2024.

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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	periov1alpha1 "github.com/openperouter/openperouter/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

// UnderlayReconciler reconciles a Underlay object
type UnderlayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Node   string
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=per.io.openperouter.github.io,resources=vnis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=per.io.openperouter.github.io,resources=vnis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=per.io.openperouter.github.io,resources=vnis/finalizers,verbs=update
// +kubebuilder:rbac:groups=per.io.openperouter.github.io,resources=underlays,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=per.io.openperouter.github.io,resources=underlays/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=per.io.openperouter.github.io,resources=underlays/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Underlay object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *UnderlayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var pods v1.PodList
	if err := r.List(ctx, &pods, client.MatchingLabels{"app": "router"},
		client.MatchingFields{
			"spec.NodeName": r.Node,
		}); err != nil {
		return ctrl.Result{}, err
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UnderlayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.NewPredicateFuncs(func(object client.Object) bool {
		switch o := object.(type) {
		case *v1.Pod:
			if o.Spec.NodeName != r.Node {
				return false
			}
			return true
		default:
			return true
		}

	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&periov1alpha1.Underlay{}).
		Watches(&v1.Pod{}, &handler.EnqueueRequestForObject{}).
		Watches(&periov1alpha1.VNI{}, &handler.EnqueueRequestForObject{}).
		WithEventFilter(p).
		Named("underlay").
		Complete(r)
}
