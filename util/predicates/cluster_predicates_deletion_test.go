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

package predicates

import (
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// A cluster that is being deleted passes the control plane predicate whatever its control plane
// looks like; a cluster that is not being deleted passes it only with an available control plane.
// This test needs no test environment.
func TestClusterWithReadyControlPlaneLetsDeletionThrough(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		name      string
		available bool
		deleting  bool
		want      bool
	}{
		{name: "control plane available", available: true, want: true},
		{name: "control plane not available", want: false},
		{name: "control plane not available, cluster being deleted", deleting: true, want: true},
		{name: "control plane available, cluster being deleted", available: true, deleting: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &clusterv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "ns"}}

			if tt.available {
				conditions.Set(cluster, metav1.Condition{
					Type:   clusterv1.ClusterControlPlaneAvailableCondition,
					Status: metav1.ConditionTrue,
					Reason: clusterv1.ReadyReason,
				})
			}

			if tt.deleting {
				cluster.DeletionTimestamp = &now
				cluster.Finalizers = []string{"capicluster.turtles.cattle.io"}
			}

			p := ClusterWithReadyControlPlane(logr.Discard())

			for kind, got := range map[string]bool{
				"update":  p.UpdateFunc(event.UpdateEvent{ObjectNew: cluster}),
				"create":  p.CreateFunc(event.CreateEvent{Object: cluster}),
				"delete":  p.DeleteFunc(event.DeleteEvent{Object: cluster}),
				"generic": p.GenericFunc(event.GenericEvent{Object: cluster}),
			} {
				if got != tt.want {
					t.Errorf("%s event: got %v, want %v", kind, got, tt.want)
				}
			}
		})
	}
}
