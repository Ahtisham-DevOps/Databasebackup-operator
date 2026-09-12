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
  targetDatabase: "postgres-secret"  # name of a Secret with host/port/username/password/database
  retentionDays: 7
```

The controller reconciles this into a running backup pipeline:
- Fetches DB credentials from the referenced Secret
- Runs `pg_dump` inside a dedicated, short-lived Kubernetes Job
- Polls the Job until it completes, updating `.status` accordingly
- Enforces retention by deleting old backup Jobs and old `.sql` files past `retentionDays`

```json
{
  "backupCount": 1,
  "lastBackupStatus": "Success",
  "lastBackupTime": "2026-09-12T00:14:29Z"
}
```

## Architecture

- **CRD**: schema-validated (`schedule`, `targetDatabase`, `retentionDays` required; `retentionDays` minimum 1)
- **Reconciler**: idempotent, cron-aware, self-requeuing (no polling on the CR itself — only briefly polls the backup Job while it runs)
- **Backup execution**: delegated to a Kubernetes Job running `postgres:16` (`pg_dump`), keeping the operator's own image minimal
- **Credentials**: read from a Kubernetes Secret at reconcile time, never stored in the CR or logged
- **Retention**: deletes old backup Jobs (owned via `OwnerReference`, so cascade-deleted with the CR) and runs a cleanup Job to remove old `.sql` files from the PVC
- **Finalizers**: guarantee cleanup runs before resource deletion
- **RBAC**: least-privilege — scoped to `databasebackups`, `secrets` (read-only), `jobs`, leader election, and metrics only
- **Metrics**: exposed on `/metrics` (authenticated, HTTPS) for Prometheus scraping

## Prerequisites

- Go 1.21+
- Docker
- `kubectl`
- A Kubernetes cluster (tested with `kind`)
- A PVC named `backup-storage` and a Secret with DB credentials in the target namespace

## Installation

```bash
make install
make docker-build IMG=databasebackup-operator:v0.2.0
kind load docker-image databasebackup-operator:v0.2.0 --name <your-cluster-name>
make deploy IMG=databasebackup-operator:v0.2.0
```

## Usage

```bash
kubectl apply -f config/samples/postgres_secret.yaml
kubectl apply -f config/samples/backup_pvc.yaml
kubectl apply -f config/samples/backup_v1_databasebackup.yaml
kubectl get databasebackup sample-backup -o jsonpath='{.status}'
```

## Testing

```bash
make test    # envtest — runs against a real fake API server
make lint    # staticcheck
```

## Design Decisions & Lessons Learned

- **Idempotency guard**: an early version updated `.status` unconditionally on every reconcile, which itself triggered a new watch event, causing an infinite reconcile loop. Fixed by only running a backup when one is actually due, based on the parsed cron schedule and `LastBackupTime`.
- **Delegating execution to a Job**: rather than running `pg_dump` inside the operator process, the operator creates a dedicated Kubernetes Job. This keeps the operator's own image small and distroless, isolates backup resource usage, and means a failed backup doesn't crash the controller.
- **Polling vs. requeue**: the controller doesn't poll the `DatabaseBackup` resource on a fixed interval — it requeues exactly at the next scheduled cron time. It does briefly poll (every 5s) while a specific backup Job is in flight, since Job completion isn't watched directly.
- **Status as a subresource**: `.status` updates use `r.Status().Update()`, kept separate from `.spec` to avoid conflicts between user edits and controller writes.
- **Running two reconcilers against the same resource** (a local `make run` instance and an in-cluster deployment) caused a stream of resource-version conflicts — a good practical reminder that only one controller should manage a given resource at a time in normal operation (or use leader election across multiple controller replicas).

## Known Limitations

- Backup files are stored on a single PVC; no S3/object storage backend yet.
- Cross-namespace Secret references are not supported (Secret must be in the same namespace as the `DatabaseBackup`).
- e2e tests are temporarily disabled pending full metrics RBAC coverage in CI.

## License

Apache 2.0
