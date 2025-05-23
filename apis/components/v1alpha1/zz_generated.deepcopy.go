//go:build !ignore_autogenerated

/*
Copyright 2025 the original author or authors.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterComponent) DeepCopyInto(out *ClusterComponent) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterComponent.
func (in *ClusterComponent) DeepCopy() *ClusterComponent {
	if in == nil {
		return nil
	}
	out := new(ClusterComponent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterComponent) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterComponentList) DeepCopyInto(out *ClusterComponentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterComponent, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterComponentList.
func (in *ClusterComponentList) DeepCopy() *ClusterComponentList {
	if in == nil {
		return nil
	}
	out := new(ClusterComponentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterComponentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Component) DeepCopyInto(out *Component) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Component.
func (in *Component) DeepCopy() *Component {
	if in == nil {
		return nil
	}
	out := new(Component)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Component) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentDuck) DeepCopyInto(out *ComponentDuck) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentDuck.
func (in *ComponentDuck) DeepCopy() *ComponentDuck {
	if in == nil {
		return nil
	}
	out := new(ComponentDuck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ComponentDuck) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentDuckList) DeepCopyInto(out *ComponentDuckList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ComponentDuck, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentDuckList.
func (in *ComponentDuckList) DeepCopy() *ComponentDuckList {
	if in == nil {
		return nil
	}
	out := new(ComponentDuckList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ComponentDuckList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentDuckSpec) DeepCopyInto(out *ComponentDuckSpec) {
	*out = *in
	out.GenericComponentSpec = in.GenericComponentSpec
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentDuckSpec.
func (in *ComponentDuckSpec) DeepCopy() *ComponentDuckSpec {
	if in == nil {
		return nil
	}
	out := new(ComponentDuckSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentDuckStatus) DeepCopyInto(out *ComponentDuckStatus) {
	*out = *in
	in.Status.DeepCopyInto(&out.Status)
	in.GenericComponentStatus.DeepCopyInto(&out.GenericComponentStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentDuckStatus.
func (in *ComponentDuckStatus) DeepCopy() *ComponentDuckStatus {
	if in == nil {
		return nil
	}
	out := new(ComponentDuckStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentList) DeepCopyInto(out *ComponentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Component, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentList.
func (in *ComponentList) DeepCopy() *ComponentList {
	if in == nil {
		return nil
	}
	out := new(ComponentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ComponentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentReference) DeepCopyInto(out *ComponentReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentReference.
func (in *ComponentReference) DeepCopy() *ComponentReference {
	if in == nil {
		return nil
	}
	out := new(ComponentReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentSpan) DeepCopyInto(out *ComponentSpan) {
	*out = *in
	if in.Trace != nil {
		in, out := &in.Trace, &out.Trace
		*out = make([]ComponentSpan, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentSpan.
func (in *ComponentSpan) DeepCopy() *ComponentSpan {
	if in == nil {
		return nil
	}
	out := new(ComponentSpan)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentSpec) DeepCopyInto(out *ComponentSpec) {
	*out = *in
	out.GenericComponentSpec = in.GenericComponentSpec
	if in.OCI != nil {
		in, out := &in.OCI, &out.OCI
		*out = new(OCIReference)
		**out = **in
	}
	if in.Ref != nil {
		in, out := &in.Ref, &out.Ref
		*out = new(ComponentReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentSpec.
func (in *ComponentSpec) DeepCopy() *ComponentSpec {
	if in == nil {
		return nil
	}
	out := new(ComponentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentStatus) DeepCopyInto(out *ComponentStatus) {
	*out = *in
	in.Status.DeepCopyInto(&out.Status)
	in.GenericComponentStatus.DeepCopyInto(&out.GenericComponentStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentStatus.
func (in *ComponentStatus) DeepCopy() *ComponentStatus {
	if in == nil {
		return nil
	}
	out := new(ComponentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Composition) DeepCopyInto(out *Composition) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Composition.
func (in *Composition) DeepCopy() *Composition {
	if in == nil {
		return nil
	}
	out := new(Composition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Composition) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CompositionDependency) DeepCopyInto(out *CompositionDependency) {
	*out = *in
	if in.Ref != nil {
		in, out := &in.Ref, &out.Ref
		*out = new(ComponentReference)
		**out = **in
	}
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(GenericConfigStoreSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.OCI != nil {
		in, out := &in.OCI, &out.OCI
		*out = new(OCIReference)
		**out = **in
	}
	if in.Composition != nil {
		in, out := &in.Composition, &out.Composition
		*out = new(GenericCompositionSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CompositionDependency.
func (in *CompositionDependency) DeepCopy() *CompositionDependency {
	if in == nil {
		return nil
	}
	out := new(CompositionDependency)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CompositionDependencyStatus) DeepCopyInto(out *CompositionDependencyStatus) {
	*out = *in
	in.WIT.DeepCopyInto(&out.WIT)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CompositionDependencyStatus.
func (in *CompositionDependencyStatus) DeepCopy() *CompositionDependencyStatus {
	if in == nil {
		return nil
	}
	out := new(CompositionDependencyStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CompositionList) DeepCopyInto(out *CompositionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Composition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CompositionList.
func (in *CompositionList) DeepCopy() *CompositionList {
	if in == nil {
		return nil
	}
	out := new(CompositionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CompositionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CompositionPlug) DeepCopyInto(out *CompositionPlug) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CompositionPlug.
func (in *CompositionPlug) DeepCopy() *CompositionPlug {
	if in == nil {
		return nil
	}
	out := new(CompositionPlug)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CompositionSpec) DeepCopyInto(out *CompositionSpec) {
	*out = *in
	out.GenericComponentSpec = in.GenericComponentSpec
	in.GenericCompositionSpec.DeepCopyInto(&out.GenericCompositionSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CompositionSpec.
func (in *CompositionSpec) DeepCopy() *CompositionSpec {
	if in == nil {
		return nil
	}
	out := new(CompositionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CompositionStatus) DeepCopyInto(out *CompositionStatus) {
	*out = *in
	in.Status.DeepCopyInto(&out.Status)
	in.GenericComponentStatus.DeepCopyInto(&out.GenericComponentStatus)
	if in.Dependencies != nil {
		in, out := &in.Dependencies, &out.Dependencies
		*out = make([]CompositionDependencyStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CompositionStatus.
func (in *CompositionStatus) DeepCopy() *CompositionStatus {
	if in == nil {
		return nil
	}
	out := new(CompositionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigStore) DeepCopyInto(out *ConfigStore) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigStore.
func (in *ConfigStore) DeepCopy() *ConfigStore {
	if in == nil {
		return nil
	}
	out := new(ConfigStore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ConfigStore) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigStoreList) DeepCopyInto(out *ConfigStoreList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ConfigStore, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigStoreList.
func (in *ConfigStoreList) DeepCopy() *ConfigStoreList {
	if in == nil {
		return nil
	}
	out := new(ConfigStoreList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ConfigStoreList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigStoreSpec) DeepCopyInto(out *ConfigStoreSpec) {
	*out = *in
	out.GenericComponentSpec = in.GenericComponentSpec
	in.GenericConfigStoreSpec.DeepCopyInto(&out.GenericConfigStoreSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigStoreSpec.
func (in *ConfigStoreSpec) DeepCopy() *ConfigStoreSpec {
	if in == nil {
		return nil
	}
	out := new(ConfigStoreSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigStoreStatus) DeepCopyInto(out *ConfigStoreStatus) {
	*out = *in
	in.Status.DeepCopyInto(&out.Status)
	in.GenericComponentStatus.DeepCopyInto(&out.GenericComponentStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigStoreStatus.
func (in *ConfigStoreStatus) DeepCopy() *ConfigStoreStatus {
	if in == nil {
		return nil
	}
	out := new(ConfigStoreStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericComponentSpec) DeepCopyInto(out *GenericComponentSpec) {
	*out = *in
	out.RepositoryRef = in.RepositoryRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericComponentSpec.
func (in *GenericComponentSpec) DeepCopy() *GenericComponentSpec {
	if in == nil {
		return nil
	}
	out := new(GenericComponentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericComponentStatus) DeepCopyInto(out *GenericComponentStatus) {
	*out = *in
	if in.WIT != nil {
		in, out := &in.WIT, &out.WIT
		*out = new(WIT)
		(*in).DeepCopyInto(*out)
	}
	if in.Trace != nil {
		in, out := &in.Trace, &out.Trace
		*out = make([]ComponentSpan, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericComponentStatus.
func (in *GenericComponentStatus) DeepCopy() *GenericComponentStatus {
	if in == nil {
		return nil
	}
	out := new(GenericComponentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericCompositionSpec) DeepCopyInto(out *GenericCompositionSpec) {
	*out = *in
	if in.Plug != nil {
		in, out := &in.Plug, &out.Plug
		*out = new(CompositionPlug)
		**out = **in
	}
	if in.Dependencies != nil {
		in, out := &in.Dependencies, &out.Dependencies
		*out = make([]CompositionDependency, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericCompositionSpec.
func (in *GenericCompositionSpec) DeepCopy() *GenericCompositionSpec {
	if in == nil {
		return nil
	}
	out := new(GenericCompositionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericConfigStoreSpec) DeepCopyInto(out *GenericConfigStoreSpec) {
	*out = *in
	if in.Values != nil {
		in, out := &in.Values, &out.Values
		*out = make([]Value, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ValuesFrom != nil {
		in, out := &in.ValuesFrom, &out.ValuesFrom
		*out = make([]ValuesFrom, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericConfigStoreSpec.
func (in *GenericConfigStoreSpec) DeepCopy() *GenericConfigStoreSpec {
	if in == nil {
		return nil
	}
	out := new(GenericConfigStoreSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCIReference) DeepCopyInto(out *OCIReference) {
	*out = *in
	out.ServiceAccountRef = in.ServiceAccountRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCIReference.
func (in *OCIReference) DeepCopy() *OCIReference {
	if in == nil {
		return nil
	}
	out := new(OCIReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Value) DeepCopyInto(out *Value) {
	*out = *in
	if in.ValueFrom != nil {
		in, out := &in.ValueFrom, &out.ValueFrom
		*out = new(ValueFrom)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Value.
func (in *Value) DeepCopy() *Value {
	if in == nil {
		return nil
	}
	out := new(Value)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ValueFrom) DeepCopyInto(out *ValueFrom) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ValueFrom.
func (in *ValueFrom) DeepCopy() *ValueFrom {
	if in == nil {
		return nil
	}
	out := new(ValueFrom)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ValuesFrom) DeepCopyInto(out *ValuesFrom) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ValuesFrom.
func (in *ValuesFrom) DeepCopy() *ValuesFrom {
	if in == nil {
		return nil
	}
	out := new(ValuesFrom)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WIT) DeepCopyInto(out *WIT) {
	*out = *in
	if in.Imports != nil {
		in, out := &in.Imports, &out.Imports
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Exports != nil {
		in, out := &in.Exports, &out.Exports
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WIT.
func (in *WIT) DeepCopy() *WIT {
	if in == nil {
		return nil
	}
	out := new(WIT)
	in.DeepCopyInto(out)
	return out
}
