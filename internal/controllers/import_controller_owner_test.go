/*
Copyright © 2023 - 2026 SUSE LLC

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
	"maps"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	managementv3 "github.com/rancher/turtles/api/rancher/management/v3"
	turtlesannotations "github.com/rancher/turtles/util/annotations"
)

func TestReconcileClusterOwner(t *testing.T) {
	const (
		creatorID = turtlesannotations.CreatorIDAnnotation
		principal = turtlesannotations.CreatorPrincipalNameAnnotation
		optOut    = turtlesannotations.NoCreatorRBACAnnotation
	)

	tests := []struct {
		name     string
		capi     map[string]string
		rancher  map[string]string
		creating bool
		want     map[string]string
	}{
		{
			name:     "new cluster, no creator named: opt out as before",
			creating: true,
			want:     map[string]string{optOut: "true"},
		},
		{
			name:     "new cluster, creator named: Rancher makes the creator owner",
			capi:     map[string]string{creatorID: "u-abc"},
			creating: true,
			want:     map[string]string{creatorID: "u-abc"},
		},
		{
			name:     "new cluster, creator and principal named: both are copied",
			capi:     map[string]string{creatorID: "u-abc", principal: "ldap://cn=abc"},
			creating: true,
			want:     map[string]string{creatorID: "u-abc", principal: "ldap://cn=abc"},
		},
		{
			name:     "new cluster, principal without creator: opt out",
			capi:     map[string]string{principal: "ldap://cn=abc"},
			creating: true,
			want:     map[string]string{optOut: "true"},
		},
		{
			name:     "new cluster, empty creator: opt out",
			capi:     map[string]string{creatorID: ""},
			creating: true,
			want:     map[string]string{optOut: "true"},
		},
		{
			name:    "existing cluster without either annotation: opt out, a creator named later is not applied",
			capi:    map[string]string{creatorID: "u-abc"},
			rancher: map[string]string{"other": "kept"},
			want:    map[string]string{"other": "kept", optOut: "true"},
		},
		{
			name:    "existing cluster that opted out stays opted out",
			capi:    map[string]string{creatorID: "u-abc"},
			rancher: map[string]string{optOut: "true"},
			want:    map[string]string{optOut: "true"},
		},
		{
			name:    "existing cluster with a creator keeps it and never gets the opt-out",
			capi:    map[string]string{creatorID: "u-other"},
			rancher: map[string]string{creatorID: "u-abc"},
			want:    map[string]string{creatorID: "u-abc"},
		},
		{
			name:    "existing cluster with a creator keeps it when the CAPI annotation is removed",
			rancher: map[string]string{creatorID: "u-abc"},
			want:    map[string]string{creatorID: "u-abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capiCluster := &clusterv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "ns", Annotations: tt.capi}}
			rancherCluster := &managementv3.Cluster{}
			rancherCluster.SetAnnotations(maps.Clone(tt.rancher))

			(&CAPIImportReconciler{}).reconcileClusterOwner(context.Background(), rancherCluster, capiCluster, tt.creating)

			if got := rancherCluster.GetAnnotations(); !maps.Equal(got, tt.want) {
				t.Errorf("annotations = %v, want %v", got, tt.want)
			}

			_, hasCreator := rancherCluster.GetAnnotations()[creatorID]
			_, hasOptOut := rancherCluster.GetAnnotations()[optOut]

			if hasCreator == hasOptOut {
				t.Errorf("exactly one of %s and %s must be set, got %v", creatorID, optOut, rancherCluster.GetAnnotations())
			}
		})
	}
}
