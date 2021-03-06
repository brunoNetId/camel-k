/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package trait

import (
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serving "knative.dev/serving/pkg/apis/serving/v1"
)

// The Owner trait ensures that all created resources belong to the integration being created
// and transfers annotations and labels on the integration onto these owned resources.
//
// +camel-k:trait=owner
type ownerTrait struct {
	BaseTrait `property:",squash"`
	// The annotations to be transferred (A comma-separated list of label keys)
	TargetAnnotations string `property:"target-annotations"`
	// The labels to be transferred (A comma-separated list of label keys)
	TargetLabels string `property:"target-labels"`
}

func newOwnerTrait() *ownerTrait {
	return &ownerTrait{
		BaseTrait: newBaseTrait("owner"),
	}
}

func (t *ownerTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if e.Integration == nil {
		return false, nil
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning), nil
}

func (t *ownerTrait) Apply(e *Environment) error {
	controller := true
	blockOwnerDeletion := true

	targetLabels := make(map[string]string)
	if e.Integration.Labels != nil {
		for _, k := range strings.Split(t.TargetLabels, ",") {
			if v, ok := e.Integration.Labels[k]; ok {
				targetLabels[k] = v
			}
		}
	}

	targetAnnotations := make(map[string]string)
	if e.Integration.Annotations != nil {
		for _, k := range strings.Split(t.TargetAnnotations, ",") {
			if v, ok := e.Integration.Annotations[k]; ok {
				targetAnnotations[k] = v
			}
		}
	}

	e.Resources.VisitMetaObject(func(res metav1.Object) {
		references := []metav1.OwnerReference{
			{
				APIVersion:         e.Integration.APIVersion,
				Kind:               e.Integration.Kind,
				Name:               e.Integration.Name,
				UID:                e.Integration.UID,
				Controller:         &controller,
				BlockOwnerDeletion: &blockOwnerDeletion,
			},
		}
		res.SetOwnerReferences(references)

		// Transfer annotations
		t.propagateLabelAndAnnotations(res, targetLabels, targetAnnotations)
	})

	e.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
		t.propagateLabelAndAnnotations(&deployment.Spec.Template, targetLabels, targetAnnotations)
	})

	e.Resources.VisitKnativeService(func(service *serving.Service) {
		t.propagateLabelAndAnnotations(&service.Spec.ConfigurationSpec.Template, targetLabels, targetAnnotations)
	})

	return nil
}

// IsPlatformTrait overrides base class method
func (t *ownerTrait) IsPlatformTrait() bool {
	return true
}

func (t *ownerTrait) propagateLabelAndAnnotations(res metav1.Object, targetLabels map[string]string, targetAnnotations map[string]string) {
	// Transfer annotations
	annotations := res.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	for k, v := range targetAnnotations {
		if _, ok := annotations[k]; !ok {
			annotations[k] = v
		}
	}
	res.SetAnnotations(annotations)

	// Transfer labels
	labels := res.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	for k, v := range targetLabels {
		if _, ok := labels[k]; !ok {
			labels[k] = v
		}
	}
	res.SetLabels(labels)
}
