package v1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	kapi "k8s.io/kubernetes/pkg/api/v1"
)

// OptionalNames is an array that may also be left nil to distinguish between set and unset.
// +protobuf.nullable=true
type OptionalNames []string

// RoleBinding references a Role, but not contain it.  It can reference any Role in the same namespace or in the global namespace.
// It adds who information via Users and Groups and namespace information by which namespace it exists in.  RoleBindings in a given
// namespace only have effect in that namespace (excepting the master namespace which has power in all namespaces).
type RoleBinding struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	kapi.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// UserNames holds all the usernames directly bound to the role
	// +genconversion=false
	UserNames OptionalNames `json:"userNames" protobuf:"bytes,2,rep,name=userNames"`
	// GroupNames holds all the groups directly bound to the role
	// +genconversion=false
	GroupNames OptionalNames `json:"groupNames" protobuf:"bytes,3,rep,name=groupNames"`
	// Subjects hold object references to authorize with this rule
	Subjects []kapi.ObjectReference `json:"subjects" protobuf:"bytes,4,rep,name=subjects"`

	// RoleRef can only reference the current namespace and the global namespace
	// If the RoleRef cannot be resolved, the Authorizer must return an error.
	// Since Policy is a singleton, this is sufficient knowledge to locate a role
	RoleRef kapi.ObjectReference `json:"roleRef" protobuf:"bytes,5,opt,name=roleRef"`
}

// RoleBindingList is a collection of RoleBindings
type RoleBindingList struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	unversioned.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of RoleBindings
	Items []RoleBinding `json:"items" protobuf:"bytes,2,rep,name=items"`
}
