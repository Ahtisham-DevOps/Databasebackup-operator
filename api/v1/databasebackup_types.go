/*
Copyright 2026.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatabaseBackupSpec defines the desired state of DatabaseBackup
type DatabaseBackupSpec struct {
	// schedule is a cron expression defining backup frequency
	// +kubebuilder:validation:Required
	Schedule string `json:"schedule"`

	// targetDatabase references the Secret containing DB connection info
	// +kubebuilder:validation:Required
	TargetDatabase string `json:"targetDatabase"`

	// retentionDays defines how many days backups are kept before deletion
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	RetentionDays int `json:"retentionDays"`
}

// DatabaseBackupStatus defines the observed state of DatabaseBackup.
type DatabaseBackupStatus struct {
	// lastBackupTime records when the most recent backup completed
	// +optional
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`

	// lastBackupStatus indicates success or failure of the most recent backup
	// +optional
	LastBackupStatus string `json:"lastBackupStatus,omitempty"`

	// backupCount tracks total successful backups performed
	// +optional
	BackupCount int `json:"backupCount,omitempty"`

	// conditions represent the current state of the DatabaseBackup resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DatabaseBackup is the Schema for the databasebackups API
type DatabaseBackup struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of DatabaseBackup
	// +required
	Spec DatabaseBackupSpec `json:"spec"`

	// status defines the observed state of DatabaseBackup
	// +optional
	Status DatabaseBackupStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// DatabaseBackupList contains a list of DatabaseBackup
type DatabaseBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []DatabaseBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &DatabaseBackup{}, &DatabaseBackupList{})
		return nil
	})
}
