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
	"log/slog"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/openperouter/openperouter/api/v1alpha1"
	periov1alpha1 "github.com/openperouter/openperouter/api/v1alpha1"
	"github.com/openperouter/openperouter/internal/pods"
	v1 "k8s.io/api/core/v1"
)

// PERouterReconciler reconciles a Underlay object
type PERouterReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	MyNode     string
	FRRConfig  string
	ReloadPort int
	PodRuntime *pods.Runtime
	LogLevel   string
	Logger     *slog.Logger
}

// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
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
func (r *PERouterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With("request", req.NamespacedName.String())
	logger.Info("controller", "UnderlayReconciler", "start reconcile")
	defer logger.Info("controller", "UnderlayReconciler", "end reconcile")

	ctx = context.WithValue(ctx, "request", req.NamespacedName.String())

	nodeIndex, err := nodeIndex(ctx, r.Client, r.MyNode)
	if err != nil {
		slog.Error("failed to fetch node index", "node", r.MyNode, "error", err)
		return ctrl.Result{}, err
	}
	routerPod, err := routerPodForNode(ctx, r.Client, r.MyNode)
	if err != nil {
		slog.Error("failed to fetch router pod", "node", r.MyNode, "error", err)
		return ctrl.Result{}, err
	}
	logger.Info("router pod", "Pod", routerPod.Name)

	var underlays v1alpha1.UnderlayList
	if err := r.Client.List(ctx, &underlays); err != nil {
		slog.Error("failed to list underlays", "error", err)
		return ctrl.Result{}, err
	}

	var vnis v1alpha1.VNIList
	if err := r.Client.List(ctx, &vnis); err != nil {
		slog.Error("failed to list vnis", "error", err)
		return ctrl.Result{}, err
	}
	logger.Debug("using config", "vnis", vnis.Items, "underlays", underlays.Items)

	if err := reloadFRRConfig(ctx, frrConfigData{
		configFile: r.FRRConfig,
		address:    routerPod.Status.PodIP,
		port:       r.ReloadPort,
		nodeIndex:  nodeIndex,
		underlays:  underlays.Items,
		logLevel:   r.LogLevel,
		vnis:       vnis.Items,
	}); err != nil {
		slog.Error("failed to reload frr config", "error", err)
		return ctrl.Result{}, err
	}

	if err := configureInterfaces(ctx, interfacesConfiguration{
		RouterPodUUID: string(routerPod.UID),
		PodRuntime:    *r.PodRuntime,
		NodeIndex:     nodeIndex,
		Underlays:     underlays.Items,
		Vnis:          vnis.Items,
	}); err != nil {
		slog.Error("failed to configure the host", "error", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PERouterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.NewPredicateFuncs(func(object client.Object) bool {
		switch o := object.(type) {
		case *v1.Pod:
			if o.Spec.NodeName != r.MyNode {
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
