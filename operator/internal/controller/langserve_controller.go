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
	"reflect"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langsmithcomv1alpha1 "github.com/nichjang/langserve-operator/api/v1alpha1"
)

// LangserveReconciler reconciles a Langserve object
type LangserveReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=langsmith.com.langchain.com,resources=langserves,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langsmith.com.langchain.com,resources=langserves/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langsmith.com.langchain.com,resources=langserves/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Langserve object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *LangserveReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	logger := log.Log.WithValues("langserveNamespace", req.NamespacedName)

	logger.Info("Starting Langserve controller reconcile")

	// fetch the langserve instance
	langserve := &langsmithcomv1alpha1.Langserve{}
	err := r.Get(ctx, req.NamespacedName, langserve)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't requeue
			logger.Info("Langserve resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Langserve instance")
		return ctrl.Result{}, err
	}

	// check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: langserve.Name, Namespace: langserve.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// create a new deployment
		deployment := r.createDeploymentFromLangserve(langserve)
		logger.Info("Creating a new deployment", "deployment.namespace", deployment.Namespace, "deployment.name", deployment.Name)
		err = r.Create(ctx, deployment)
		if err != nil {
			logger.Error(err, "Error creating new deployment", "deployment.namespace", deployment.Namespace, "deployment.name", deployment.Name)
			return ctrl.Result{}, err
		}
		// deployment created, return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "error getting deployment")
		// reconcile failed due to error
		return ctrl.Result{}, err
	}

	// Deployment has been created at this point
	// Make sure the replicas match the spec
	replicas := langserve.Spec.Replicas
	if *found.Spec.Replicas != replicas {
		found.Spec.Replicas = &replicas
		err = r.Update(ctx, found)
		if err != nil {
			logger.Error(err, "error updating deployment", "deployment.namespace", found.Namespace, "deployment.name", found.Name)
			return ctrl.Result{}, err
		}
		// deployment spec updated, return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	// Update the Langserve status with pod names
	// List the pods for this Langserve's deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(langserve.Namespace),
		client.MatchingLabels(langserve.GetLabels()),
	}

	if err = r.List(ctx, podList, listOpts...); err != nil {
		logger.Error(err, "falied to list pods", "langserve.namespace", langserve.Namespace, "langserve.name", langserve.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update langserve.Status.Pods if needed
	if !reflect.DeepEqual(podNames, langserve.Status.Pods) {
		langserve.Status.Pods = podNames
		err := r.Update(ctx, langserve)
		if err != nil {
			logger.Error(err, "Failed to update InhouseApp status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LangserveReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langsmithcomv1alpha1.Langserve{}).
		Complete(r)
}

func (r *LangserveReconciler) createDeploymentFromLangserve(m *langsmithcomv1alpha1.Langserve) *appsv1.Deployment {
	ls := labelsForLangserve(m.Spec.Image, m.Spec.Name)
	replicas := m.Spec.Replicas

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: m.Spec.Image,
						Name:  m.Spec.Name,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Name:          "http",
						}},
					}},
				},
			},
		},
	}
	ctrl.SetControllerReference(m, deploy, r.Scheme)
	return deploy
}

func labelsForLangserve(image string, name string) map[string]string {
	var imageTag string
	if strings.Contains(image, ":") {
		imageTag = strings.Split(image, ":")[1]
	}
	return map[string]string{"app.kubernetes.io/name": "Memcached",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/version":    imageTag,
		"app.kubernetes.io/part-of":    "memcached-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
