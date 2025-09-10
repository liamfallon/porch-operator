/*
Copyright 2025.

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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true

// PackageRevisionList contains a list of PackageRevision.
type PackageRevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PackageRevision `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PackageRevision is the Schema for the packagerevisions API.
type PackageRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PackageRevisionSpec   `json:"spec,omitempty"`
	Status PackageRevisionStatus `json:"status,omitempty"`
}

// PackageRevisionSpec defines the desired state of PackageRevision.
type PackageRevisionSpec struct {
	// PackageName identifies the package in the repository.
	PackageName string `json:"packageName,omitempty"`

	// RepositoryName is the name of the Repository object containing this package.
	RepositoryName string `json:"repository,omitempty"`

	// WorkspaceName is a short, unique description of the changes contained in this package revision.
	WorkspaceName string `json:"workspaceName,omitempty"`

	// Revision identifies the version of the package.
	// +kubebuilder:validation:Minimum=-1
	Revision int `json:"revision,omitempty"`

	// Parent references a package that provides resources to us
	Parent *ParentReference `json:"parent,omitempty"`

	Lifecycle PackageRevisionLifecycle `json:"lifecycle,omitempty"`

	Tasks []Task `json:"tasks,omitempty"`

	ReadinessGates []ReadinessGate `json:"readinessGates,omitempty"`
}

// PackageRevisionStatus defines the observed state of PackageRevision.
type PackageRevisionStatus struct {
	// Represents the observations of a Memcached's current state.
	// Memcached.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// Memcached.status.conditions.status are one of True, False, Unknown.
	// Memcached.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// Memcached.status.conditions.Message is a human readable message indicating details about the transition.
	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// UpstreamLock identifies the upstream data for this package.
	UpstreamLock *UpstreamLock `json:"upstreamLock,omitempty"`

	// PublishedBy is the identity of the user who approved the packagerevision.
	PublishedBy string `json:"publishedBy,omitempty"`

	// PublishedAt is the time when the packagerevision were approved.
	PublishedAt metav1.Time `json:"publishTimestamp,omitempty"`

	// Deployment is true if this is a deployment package (in a deployment repository).
	Deployment bool `json:"deployment,omitempty"`

	// Conditions store the status conditions of the Memcached instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// ParentReference is a reference to a parent package
type ParentReference struct {
	// TODO: Should this be a revision or a package?

	// Name is the name of the parent PackageRevision
	Name string `json:"name"`
}

type PackageRevisionLifecycle string

const (
	PackageRevisionLifecycleDraft            PackageRevisionLifecycle = "Draft"
	PackageRevisionLifecycleProposed         PackageRevisionLifecycle = "Proposed"
	PackageRevisionLifecyclePublished        PackageRevisionLifecycle = "Published"
	PackageRevisionLifecycleDeletionProposed PackageRevisionLifecycle = "DeletionProposed"
)

type Task struct {
	Type    TaskType                `json:"type"`
	Init    *PackageInitTaskSpec    `json:"init,omitempty"`
	Clone   *PackageCloneTaskSpec   `json:"clone,omitempty"`
	Edit    *PackageEditTaskSpec    `json:"edit,omitempty"`
	Upgrade *PackageUpgradeTaskSpec `json:"upgrade,omitempty"`
}

type TaskType string

const (
	TaskTypeInit    TaskType = "init"
	TaskTypeClone   TaskType = "clone"
	TaskTypeEdit    TaskType = "edit"
	TaskTypeUpgrade TaskType = "upgrade"
)

type ReadinessGate struct {
	ConditionType string `json:"conditionType,omitempty"`
}

type PackageInitTaskSpec struct {
	// `Subpackage` is a directory path to a subpackage to initialize. If unspecified, the main package will be initialized.
	Subpackage string `json:"subpackage,omitempty"`
	// `Description` is a short description of the package.
	Description string `json:"description,omitempty"`
	// `Keywords` is a list of keywords describing the package.
	Keywords []string `json:"keywords,omitempty"`
	// `Site is a link to page with information about the package.
	Site string `json:"site,omitempty"`
}

type PackageCloneTaskSpec struct {
	// // `Subpackage` is a path to a directory where to clone the upstream package.
	// Subpackage string `json:"subpackage,omitempty"`

	// `Upstream` is the reference to the upstream package to clone.
	Upstream UpstreamPackage `json:"upstreamRef,omitempty"`

	// 	Defines which strategy should be used to update the package. It defaults to 'resource-merge'.
	//  * resource-merge: Perform a structural comparison of the original /
	//    updated resources, and merge the changes into the local package.
	//  * fast-forward: Fail without updating if the local package was modified
	//    since it was fetched.
	//  * force-delete-replace: Wipe all the local changes to the package and replace
	//    it with the remote version.
	//  * copy-merge: Copy all the remote changes to the local package.
	Strategy PackageMergeStrategy `json:"strategy,omitempty"`
}

type PackageUpgradeTaskSpec struct {
	// `OldUpstream` is the reference to the original upstream package revision that is
	// the common ancestor of the local package and the new upstream package revision.
	OldUpstream PackageRevisionRef `json:"oldUpstreamRef,omitempty"`

	// `NewUpstream` is the reference to the new upstream package revision that the
	// local package will be upgraded to.
	NewUpstream PackageRevisionRef `json:"newUpstreamRef,omitempty"`

	// `LocalPackageRevisionRef` is the reference to the local package revision that
	// contains all the local changes on top of the `OldUpstream` package revision.
	LocalPackageRevisionRef PackageRevisionRef `json:"localPackageRevisionRef,omitempty"`

	// 	Defines which strategy should be used to update the package. It defaults to 'resource-merge'.
	//  * resource-merge: Perform a structural comparison of the original /
	//    updated resources, and merge the changes into the local package.
	//  * fast-forward: Fail without updating if the local package was modified
	//    since it was fetched.
	//  * force-delete-replace: Wipe all the local changes to the package and replace
	//    it with the remote version.
	//  * copy-merge: Copy all the remote changes to the local package.
	Strategy PackageMergeStrategy `json:"strategy,omitempty"`
}

type PackageEditTaskSpec struct {
	Source *PackageRevisionRef `json:"sourceRef,omitempty"`
}

type UpstreamPackage struct {
	// Type of the repository (i.e. git, OCI). If empty, `upstreamRef` will be used.
	Type RepositoryType `json:"type,omitempty"`

	// Git upstream package specification. Required if `type` is `git`. Must be unspecified if `type` is not `git`.
	Git *GitPackage `json:"git,omitempty"`

	// OCI upstream package specification. Required if `type` is `oci`. Must be unspecified if `type` is not `oci`.
	Oci *OciPackage `json:"oci,omitempty"`

	// UpstreamRef is the reference to the package from a registered repository rather than external package.
	UpstreamRef *PackageRevisionRef `json:"upstreamRef,omitempty"`
}

type RepositoryType string

const (
	RepositoryTypeGit RepositoryType = "git"
	RepositoryTypeOCI RepositoryType = "oci"
)

type GitPackage struct {
	// Address of the Git repository, for example:
	//   `https://github.com/GoogleCloudPlatform/blueprints.git`
	Repo string `json:"repo"`

	// `Ref` is the git ref containing the package. Ref can be a branch, tag, or commit SHA.
	Ref string `json:"ref"`

	// Directory within the Git repository where the packages are stored. A subdirectory of this directory containing a Kptfile is considered a package.
	Directory string `json:"directory"`

	// Reference to secret containing authentication credentials. Optional.
	SecretRef SecretRef `json:"secretRef,omitempty"`
}

type SecretRef struct {
	// Name of the secret. The secret is expected to be located in the same namespace as the resource containing the reference.
	Name string `json:"name"`
}

// OciPackage describes a repository compatible with the Open Container Registry standard.
type OciPackage struct {
	// Image is the address of an OCI image.
	Image string `json:"image"`
}

type PackageRevisionRef struct {
	// `Name` is the name of the referenced PackageRevision resource.
	Name string `json:"name"`
}

type PackageMergeStrategy string

const (
	ResourceMerge      PackageMergeStrategy = "resource-merge"
	FastForward        PackageMergeStrategy = "fast-forward"
	ForceDeleteReplace PackageMergeStrategy = "force-delete-replace"
	CopyMerge          PackageMergeStrategy = "copy-merge"
)

type UpstreamLock struct {
	// Type is the type of origin.
	Type OriginType `json:"type,omitempty"`

	// Git is the resolved locator for a package on Git.
	Git *GitLock `json:"git,omitempty"`
}

type GitLock struct {
	// Repo is the git repository that was fetched.
	// e.g. 'https://github.com/kubernetes/examples.git'
	Repo string `json:"repo,omitempty"`

	// Directory is the sub directory of the git repository that was fetched.
	// e.g. 'staging/cockroachdb'
	Directory string `json:"directory,omitempty"`

	// Ref can be a Git branch, tag, or a commit SHA-1 that was fetched.
	// e.g. 'master'
	Ref string `json:"ref,omitempty"`

	// Commit is the SHA-1 for the last fetch of the package.
	// This is set by kpt for bookkeeping purposes.
	Commit string `json:"commit,omitempty"`
}

type OriginType string

func init() {
	SchemeBuilder.Register(&PackageRevision{}, &PackageRevisionList{})
}
