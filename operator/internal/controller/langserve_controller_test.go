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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	langsmithcomv1alpha1 "github.com/nichjang/langserve-operator/api/v1alpha1"
)

var _ = Describe("Langserve controller", func() {
	Context("Langserve controller test", func() {

		const LangserveName = "test-langserve"

		ctx := context.Background()

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      LangserveName,
				Namespace: LangserveName,
			},
		}

		typeNamespaceName := types.NamespacedName{Name: LangserveName, Namespace: LangserveName}

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, namespace)
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterEach(func() {
			// TODO(user): Attention if you improve this code by adding other context test you MUST
			// be aware of the current delete namespace limitations. More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Deleting the Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, namespace)
		})

		It("should successfully reconcile a custom resource for Langserve", func() {
			By("Creating the custom resource for the Kind Langserve")
			langserve := &langsmithcomv1alpha1.Langserve{}
			err := k8sClient.Get(ctx, typeNamespaceName, langserve)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				langserve := &langsmithcomv1alpha1.Langserve{
					ObjectMeta: metav1.ObjectMeta{
						Name:      LangserveName,
						Namespace: namespace.Name,
					},
					Spec: langsmithcomv1alpha1.LangserveSpec{
						Replicas: 3,
						Image:    "test-image",
						Name:     "test-name",
					},
				}

				err = k8sClient.Create(ctx, langserve)
				Expect(err).To(Not(HaveOccurred()))
			}

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &langsmithcomv1alpha1.Langserve{}
				return k8sClient.Get(ctx, typeNamespaceName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			langserveReconciler := &LangserveReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = langserveReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func() error {
				found := &appsv1.Deployment{}
				return k8sClient.Get(ctx, typeNamespaceName, found)
			}, time.Minute, time.Second).Should(Succeed())
		})
	})
})
