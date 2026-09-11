# DatabaseBackup Operator

A production-grade Kubernetes Operator for automating scheduled database backups, built with Go, kubebuilder, and controller-runtime.

## What It Does

`DatabaseBackup` is a Custom Resource that lets you declaratively manage database backup schedules on Kubernetes:

```yaml
apiVersion: backup.example.com/v1
kind: DatabaseBackup
metadata:
  name: sample-backup
spec:
  schedule: "0 2 * * *"        # cron expression
  targetDatabase: "postgres-secret"
  retentionDays: 7
```

The controller reconciles this into a running backup schedule, tracking status in `.status`:

```json
{
  "backupCount": 1,
  "lastBackupStatus": "Success",
  "lastBackupTime": "2026-09-10T17:15:15Z"
}
```

## Architecture

- **CRD**: schema-validated (`schedule`, `targetDatabase`, `retentionDays` required; `retentionDays` minimum 1)
- **Reconciler**: idempotent, cron-aware, self-requeuing (no polling — waits until the next scheduled run)
- **Finalizers**: guarantee cleanup runs before resource deletion
- **RBAC**: least-privilege — scoped to `databasebackups` resources, leader election, and metrics only
- **Metrics**: exposed on `/metrics` (authenticated, HTTPS) for Prometheus scraping

## Prerequisites

- Go 1.21+
- Docker
- `kubectl`
- A Kubernetes cluster (tested with `kind`)

## Installation

```bash
# Install the CRD
make install

# Build and load the image (for local kind clusters)
make docker-build IMG=databasebackup-operator:v0.1.0
kind load docker-image databasebackup-operator:v0.1.0 --name <your-cluster-name>

# Deploy
make deploy IMG=databasebackup-operator:v0.1.0
```

## Usage

```bash
kubectl apply -f config/samples/backup_v1_databasebackup.yaml
kubectl get databasebackup sample-backup -o jsonpath='{.status}'
```

## Testing

```bash
make test    # envtest — runs against a real fake API server, no live cluster needed
make lint    # staticcheck
```

## Design Decisions & Lessons Learned

- **Idempotency guard**: an early version of the reconciler updated `.status` unconditionally on every reconcile, which itself triggered a new watch event — causing an infinite reconcile loop. Fixed by adding a guard that only runs a backup when one is actually due, based on the parsed cron schedule and `LastBackupTime`.
- **Explicit requeue over polling**: instead of reconciling on a fixed interval, the controller calculates the next scheduled run and requeues exactly then (`RequeueAfter`), avoiding unnecessary API server load.
- **Status as a subresource**: `.status` updates use `r.Status().Update()`, kept separate from `.spec` updates to avoid conflicts between user edits and controller writes.

## Known Limitations

- Backup execution is currently simulated (no real `pg_dump`/`mysqldump` integration yet).
- Retention policy (`retentionDays`) enforcement is not yet implemented — pending real backup storage integration.
- e2e tests are temporarily disabled pending full metrics RBAC coverage in CI.

## License

Apache 2.0
