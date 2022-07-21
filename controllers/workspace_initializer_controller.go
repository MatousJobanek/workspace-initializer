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
	"time"

	kcpclient "github.com/kcp-dev/apimachinery/pkg/client"
	"github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
	v1alpha12 "github.com/kcp-dev/kcp/pkg/apis/tenancy/v1alpha1"
	"github.com/kcp-dev/logicalcluster"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// WorkspaceInitializer reconciles a ClusterWorkspace object
type WorkspaceInitializer struct {
	Client client.Client
	Scheme *runtime.Scheme
	Config *rest.Config
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkspaceInitializer) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha12.ClusterWorkspace{}).
		Complete(r)
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

	workspaceCtx := kcpclient.WithCluster(ctx, logicalcluster.New(req.ClusterName))
	workspace := &v1alpha12.ClusterWorkspace{}
	if err := r.Client.Get(workspaceCtx, req.NamespacedName, workspace); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	clusterName := fmt.Sprintf("%s:%s", req.ClusterName, req.Name)
	logger.Info("reconciling for clusterName", "clusterName", clusterName)
	cfg := rest.CopyConfig(r.Config)
	cfg.Host = fmt.Sprintf("%s/clusters/%s", cfg.Host, clusterName)
	cl, err := client.New(cfg, client.Options{
		Scheme: r.Scheme,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	ctx = kcpclient.WithCluster(ctx, logicalcluster.New(clusterName))

	apiBinding := &v1alpha1.APIBinding{
		TypeMeta: v1.TypeMeta{
			Kind:       "APIBinding",
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:        "crcworkload",
			ClusterName: clusterName,
		},
		Spec: v1alpha1.APIBindingSpec{
			Reference: v1alpha1.ExportReference{
				Workspace: &v1alpha1.WorkspaceExportReference{
					ExportName: "kubernetes",
					Path:       "root:synctarget",
				},
			},
		},
	}
	fmt.Println("creating first APIBinding")
	if err := cl.Create(ctx, apiBinding); err != nil {
		if !errors.IsAlreadyExists(err) {
			return ctrl.Result{}, err
		}
	}

	apiBinding = &v1alpha1.APIBinding{
		TypeMeta: v1.TypeMeta{
			Kind:       "APIBinding",
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:        "has",
			ClusterName: clusterName,
		},
		Spec: v1alpha1.APIBindingSpec{
			Reference: v1alpha1.ExportReference{
				Workspace: &v1alpha1.WorkspaceExportReference{
					ExportName: "application-service-has",
					Path:       "root:has",
				},
			},
		},
	}

	fmt.Println("creating second APIBinding")
	if err := cl.Create(ctx, apiBinding); err != nil {
		if !errors.IsAlreadyExists(err) {
			return ctrl.Result{}, err
		}
	} else {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
	}
	//for i, init := range workspace.Status.Initializers {
	//	if init == "root:plane:usersignup:appstudio" {
	//		workspace.Status.Initializers = append(workspace.Status.Initializers[:i], workspace.Status.Initializers[i+1:]...)
	//		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, r.Client.Status().Update(workspaceCtx, workspace)
	//	}
	//}

	//
	workspace.Status.Initializers = nil
	fmt.Println("removing all initializers because of the bug in kcp")
	return ctrl.Result{}, r.Client.Status().Update(workspaceCtx, workspace)
}
