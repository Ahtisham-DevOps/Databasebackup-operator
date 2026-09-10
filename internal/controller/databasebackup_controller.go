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

package controller

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	backupv1 "github.com/Ahtisham-DevOps/databasebackup-operator/api/v1"
)

const databaseBackupFinalizer = "backup.example.com/finalizer"

// DatabaseBackupReconciler reconciles a DatabaseBackup object
type DatabaseBackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.example.com,resources=databasebackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.example.com,resources=databasebackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.example.com,resources=databasebackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DatabaseBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile
func (r *DatabaseBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var backup backupv1.DatabaseBackup
	if err := r.Get(ctx, req.NamespacedName, &backup); err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Info("DatabaseBackup deleted, nothing to do", "name", req.Name)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch DatabaseBackup")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !backup.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&backup, databaseBackupFinalizer) {
			log.Info("Cleaning up before delete", "name", backup.Name)
			// TODO: real cleanup (delete stored backup files) yahan aayega

			controllerutil.RemoveFinalizer(&backup, databaseBackupFinalizer)
			if err := r.Update(ctx, &backup); err != nil {
				log.Error(err, "unable to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&backup, databaseBackupFinalizer) {
		controllerutil.AddFinalizer(&backup, databaseBackupFinalizer)
		if err := r.Update(ctx, &backup); err != nil {
			log.Error(err, "unable to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	schedule, err := cron.ParseStandard(backup.Spec.Schedule)
	if err != nil {
		log.Error(err, "invalid cron schedule", "schedule", backup.Spec.Schedule)
		return ctrl.Result{}, nil
	}

	now := time.Now()

	if backup.Status.LastBackupTime == nil {
		return r.runBackup(ctx, &backup, now)
	}

	nextRun := schedule.Next(backup.Status.LastBackupTime.Time)

	if now.Before(nextRun) {
		log.Info("Waiting for next scheduled backup", "nextRun", nextRun)
		return ctrl.Result{RequeueAfter: nextRun.Sub(now)}, nil
	}

	return r.runBackup(ctx, &backup, now)
}

func (r *DatabaseBackupReconciler) runBackup(ctx context.Context, backup *backupv1.DatabaseBackup, now time.Time) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Running backup", "targetDatabase", backup.Spec.TargetDatabase)

	backupTime := metav1.NewTime(now)
	backup.Status.LastBackupTime = &backupTime
	backup.Status.LastBackupStatus = "Success"
	backup.Status.BackupCount = backup.Status.BackupCount + 1

	if err := r.Status().Update(ctx, backup); err != nil {
		log.Error(err, "unable to update DatabaseBackup status")
		return ctrl.Result{}, err
	}

	log.Info("Backup status updated", "backupCount", backup.Status.BackupCount)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.DatabaseBackup{}).
		Named("databasebackup").
		Complete(r)
}
