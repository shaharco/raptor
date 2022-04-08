/*
Copyright 2022 Natun.

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

package v1alpha1

// Important: Run "make" to regenerate code after modifying this file

import (
	"fmt"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AggrType defines the type of aggregation
// +kubebuilder:validation:Enum=count;min;max;sum;avg;mean
type AggrType string

// FeatureSpec defines the desired state of Feature
type FeatureSpec struct {
	// Primitive defines the type of the underlying feature-value that a Feature should respond with
	// Valid values are:
	//  - `int`
	//  - `float`
	//  - `string`
	//	- `timestamp
	//  - `[]int`
	//  - `[]float`
	//  - `[]string`
	//  - `[]timestamp`
	//  - `headless`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=int;float;string;timestamp;[]int;[]float;[]string;[]timestamp;headless
	Primitive string `json:"primitive"`

	// Freshness defines the age of a feature-value(time since the value has set) to consider as *fresh*.
	// Fresh values doesn't require re-ingestion
	// +kubebuilder:validation:Required
	Freshness metav1.Duration `json:"freshness"`

	// Staleness defines the age of a feature-value(time since the value has set) to consider as *stale*.
	// Stale values are not fit for usage, therefore will not be returned and will REQUIRE re-ingestion.
	// +kubebuilder:validation:Required
	Staleness metav1.Duration `json:"staleness"`

	// Timeout defines the maximum ingestion time allowed to calculate the feature value.
	// +optional
	// +nullable
	Timeout metav1.Duration `json:"timeout"`

	// Aggr defines an aggregation on top of the underlying feature-value. Aggregations will be calculated on time-of-request.
	// Users can specify here multiple functions to calculate the aggregation.
	// Valid values:
	//  - `count`
	//  - `min`
	//  - `mix`
	//  - `sum``
	//  - `mean` (alias for `avg`)
	//  - `avg`
	// +optional
	// +nullable
	Aggr []AggrType `json:"aggr"`

	// Builder defines a building-block to use to build the feature-value
	// +kubebuilder:validation:Required
	Builder FeatureBuilder `json:"builder"`
}

// FeatureBuilderType select the building-block to use to build the feature-value
type FeatureBuilderType struct {
	// Kind defines the type of Builder to use to build the feature-value.
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`
}

// FeatureBuilder defines a building-block to use to build the feature-value
type FeatureBuilder struct {
	FeatureBuilderType `json:",inline"`

	// Embedded custom configuration of the Builder to use to build the feature-value.
	apiextensions.JSON `json:",inline"`
}

// FeatureStatus defines the observed state of Feature
type FeatureStatus struct {
	// FQN is the Fully Qualified Name for the feature
	FQN string `json:"fqn"`

	// Connector is a reference for the DataConnector that this Feature is associated with
	Connector ResourceReference `json:"connector"`
}

// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=datascience,shortName=ft

// Feature is the Schema for the features API
type Feature struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FeatureSpec   `json:"spec,omitempty"`
	Status FeatureStatus `json:"status,omitempty"`
}

// FQN returns the fully qualified name of the feature.
func (in *Feature) FQN() string {
	return fmt.Sprintf("%s.%s", in.GetName(), in.GetNamespace())
}

//+kubebuilder:object:root=true

// FeatureList contains a list of Feature
type FeatureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Feature `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Feature{}, &FeatureList{})
}
