/*
Copyright 2023.

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
	helmstorage "helm.sh/helm/v3/pkg/storage"
	helmstoragedriver "helm.sh/helm/v3/pkg/storage/driver"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// FooReconciler reconciles a Foo object
type FooReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
}

//+kubebuilder:rbac:groups=my.domain,resources=foos,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;
//+kubebuilder:rbac:groups=my.domain,resources=foos/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my.domain,resources=foos/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Foo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *FooReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	fmt.Println("Reconcile is invoked !!!!!")

	_ = log.FromContext(ctx)

	clientset, err := kubernetes.NewForConfig(r.RestConfig)
	if err != nil {
		panic(err)
	}

	storage := helmstorage.Init(helmstoragedriver.NewSecrets(clientset.CoreV1().Secrets("")))
	releases, err := storage.ListReleases()
	if err != nil {
		panic(err)
	}

	for _, release := range releases {
		fmt.Println(release.Chart.Metadata.Annotations)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FooReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&appsv1.Deployment{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				labels := e.ObjectNew.GetLabels()
				if labels["app.kubernetes.io/managed-by"] == "Helm" {
					e.ObjectOld.GetGeneration()
					fmt.Println("UpdateFunc found match", e.ObjectOld.GetResourceVersion(), e.ObjectNew.GetResourceVersion())
					return true
				} else {
					fmt.Println("UpdateFunc no match")
					return false
				}
			},
			CreateFunc: func(e event.CreateEvent) bool {
				labels := e.Object.GetLabels()
				if labels["app.kubernetes.io/managed-by"] == "Helm" {
					fmt.Println("CreateFunc found match")
					return true
				} else {
					fmt.Println("CreateFunc no match")
					return false
				}
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				labels := e.Object.GetLabels()
				if labels["app.kubernetes.io/managed-by"] == "Helm" {
					fmt.Println("DeleteFunc found match")
					return true
				} else {
					fmt.Println("DeleteFunc no match")
					return false
				}
			},
		}).
		Complete(r)
}
