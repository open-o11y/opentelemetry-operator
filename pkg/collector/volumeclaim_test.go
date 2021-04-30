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

package collector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

func TestVolumeClaimNewDefault(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{}
	cfg := config.New()

	// test
	volumeClaims := VolumeClaimTemplates(cfg, otelcol)

	// verify
	assert.Len(t, volumeClaims, 1)

	// check that it's the initial-volume
	assert.Equal(t, "initial-volume", volumeClaims[0].Name)

	// check the access mode is correct
	assert.Equal(t, v1.PersistentVolumeAccessMode("ReadOnlyMany"), volumeClaims[0].Spec.AccessModes[0])

	//check the storage is correct
	assert.Equal(t, resource.MustParse("50Mi"), volumeClaims[0].Spec.Resources.Requests["storage"])
}

func TestVolumeClaimAllowsUserToAdd(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "added-volume",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{"ReadOnlyMany"},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{"storage": resource.MustParse("1Gi")},
					},
				},
			}},
		},
	}
	cfg := config.New()

	// test
	volumeClaims := VolumeClaimTemplates(cfg, otelcol)

	// verify that volume claim replaces
	assert.Len(t, volumeClaims, 1)

	// check that it's the added volume
	assert.Equal(t, "added-volume", volumeClaims[0].Name)

	// check the access mode is correct
	assert.Equal(t, v1.PersistentVolumeAccessMode("ReadOnlyMany"), volumeClaims[0].Spec.AccessModes[0])

	//check the storage is correct
	assert.Equal(t, resource.MustParse("1Gi"), volumeClaims[0].Spec.Resources.Requests["storage"])
}
