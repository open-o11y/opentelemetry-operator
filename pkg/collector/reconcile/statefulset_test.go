// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reconcile_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/reconcile"
)

var logger = logf.Log.WithName("unit-tests")

func TestExpectedStatefulSets(t *testing.T) {
	// prepare
	var prev_replicas int32 = 3
	var new_replicas int32 = 4
	cfg := config.New()
	nsn := types.NamespacedName{Name: "existingstatefulset", Namespace: "default"}
	labels := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
	})

	// First create a StatefulSet instance in the client as the "existing" instance
	existed := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &prev_replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
			},
		},
	}

	err := k8sClient.Create(context.Background(), existed)
	require.NoError(t, err)

	existing := &appsv1.StatefulSet{}
	err = k8sClient.Get(context.Background(), nsn, existing)
	require.NoError(t, err)
	// Make sure the existing StatefulSet has a Replica of 3.
	assert.Equal(t, prev_replicas, *existing.Spec.Replicas)

	// Create a fake OpenTelemetryCollector object so we can
	// pass in Name and UID of the existing StatefulSet into
	// ExpectedStatefulSets()
	instance := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			UID:       existing.UID,
		},
	}
	params := reconcile.Params{
		Client:   k8sClient,
		Log:      logger,
		Scheme:   testScheme,
		Config:   cfg,
		Instance: instance,
	}

	// We will pass in two StatefulSets to ExpectedStatefulSets().
	// One will update the existing StatefulSet's replica to 4, and
	// the other will create a new StatefulSet.
	update := []appsv1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &new_replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "newstatefulset",
				Namespace: nsn.Namespace,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &new_replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
				},
			},
		},
	}

	err = reconcile.ExpectedStatefulSets(context.Background(), params, update)
	require.NoError(t, err)

	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
	}

	list := &appsv1.StatefulSetList{}
	err = k8sClient.List(context.Background(), list, opts...)
	assert.NoError(t, err)

	assert.Len(t, list.Items, 2)
	assert.Equal(t, nsn.Name, list.Items[0].Name)
	// The existing StatefulSet now should have a Replica of 4.
	assert.Equal(t, new_replicas, *list.Items[0].Spec.Replicas)

	assert.Equal(t, "newstatefulset", list.Items[1].Name)
	assert.Equal(t, new_replicas, *list.Items[1].Spec.Replicas)

	// cleanup
	require.NoError(t, k8sClient.Delete(context.Background(), &list.Items[0]))
	require.NoError(t, k8sClient.Delete(context.Background(), &list.Items[1]))

}
