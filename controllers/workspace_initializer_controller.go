/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"

	kcpclient "github.com/kcp-dev/apimachinery/pkg/client"
	"github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
	v1alpha12 "github.com/kcp-dev/kcp/pkg/apis/tenancy/v1alpha1"
	"github.com/kcp-dev/logicalcluster"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// WorkspaceInitializer reconciles a ClusterWorkspace object
type WorkspaceInitializer struct {
	Client    client.Client
	Scheme    *runtime.Scheme
	Namespace string
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkspaceInitializer) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha12.ClusterWorkspace{})
	return builder.Complete(r)
}

//+kubebuilder:rbac:groups=tenancy.kcp.dev,resources=clusterworkspaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tenancy.kcp.dev,resources=clusterworkspaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tenancy.kcp.dev,resources=clusterworkspaces/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterWorkspace object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *WorkspaceInitializer) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconciling")

	clusterName := fmt.Sprintf("%s:%s", req.ClusterName, req.Name)
	logger.Info("reconciling for clusterName", "clusterName", clusterName)

	//ctx = kcpclient.WithCluster(ctx, logicalcluster.New(req.ClusterName))
	//
	//cw := &v1alpha12.ClusterWorkspace{}
	//if err := r.Client.Get(ctx, req.NamespacedName, cw); err != nil {
	//	return ctrl.Result{}, err
	//}
	ctx = kcpclient.WithCluster(ctx, logicalcluster.New(clusterName))

	apiBinding := &v1alpha1.APIBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:        "workload",
			ClusterName: clusterName,
		},
		Spec: v1alpha1.APIBindingSpec{
			Reference: v1alpha1.ExportReference{
				Workspace: &v1alpha1.WorkspaceExportReference{
					ExportName: "kubernetes",
					Path:       "root:plane:crc",
				},
			},
		},
	}

	return ctrl.Result{}, r.Client.Create(ctx, apiBinding)
	//bindings := &v1alpha1.APIBindingList{}
	//r.Client.List(context.TODO())
	//err := r.Client().Get(context.TODO(), req.NamespacedName, workspace)
	//if err != nil {
	//	if errors.IsNotFound(err) {
	//		// Request object not found, could have been deleted after reconcile request.
	//		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
	//		// Return and don't requeue
	//		return reconcile.Result{}, nil
	//	}
	//	// Error reading the object - requeue the request.
	//	return reconcile.Result{}, err
	//}
	//if workspace.Status.Phase != v1alpha12.ClusterWorkspacePhaseReady || workspace.Status.BaseURL == "" {
	//	logger.Info("workspace is not ready yet")
	//	return ctrl.Result{}, nil
	//}
	//if workspace.Spec.Type != "Universal" {
	//	logger.Info("workspace is not of universal type")
	//	return ctrl.Result{}, nil
	//}
	//if workspace.Labels["type"] != "appstudio" {
	//	logger.Info("workspace is not an appstudio workspace")
	//	return ctrl.Result{}, nil
	//}
	//
	//return ctrl.Result{}, nil
}
