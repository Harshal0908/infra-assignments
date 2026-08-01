# Kubernetes Config Service

A small Go HTTP service that stores configuration records in PostgreSQL and runs in a local Kind Kubernetes cluster.

## Architecture

```text
Local client
    |
    | kubectl port-forward
    v
Kubernetes Service
    |
    v
Go Application Deployment
    |
    | SQL
    v
PostgreSQL Service
    |
    v
PostgreSQL StatefulSet
    |
    v
PersistentVolumeClaim
```

Resource ownership:

```text
Terraform
├── Kubernetes namespace
├── ConfigMap
├── Secret
├── PostgreSQL Service
├── PostgreSQL StatefulSet
└── PersistentVolumeClaim

Kubernetes manifests
├── Database migration Job
├── Application Deployment
└── Application Service
```

## Repository Structure

```text
cmd/                     Application entry point
internal/handler/        HTTP handlers
internal/service/        Validation and application rules
internal/repository/     Repository interface
internal/repository/postgres/
                         PostgreSQL implementation
internal/domain/         Shared data types
migrations/              Database schema
infra/terraform/         Infrastructure as code
k8s/                     Kubernetes application resources
scripts/                 Smoke-test automation
docs/                    Design documentation
```

## Prerequisites

Install:

* Go 1.25 or newer
* Docker
* Kind
* `kubectl`
* Terraform 1.8 or newer
* Make
* `curl`

Docker must be running.

## Quick Start

From this directory, run:

```bash
make bootstrap
```

This command:

1. runs Go tests;
2. creates the Kind cluster if needed;
3. builds the application image;
4. loads the image into Kind;
5. provisions infrastructure with Terraform;
6. runs the database migration;
7. deploys the application.

Check the deployed resources:

```bash
make status
```

Run the automated API validation:

```bash
make validate
```

Expected result:

```text
All smoke tests passed.
```

## API

### Liveness

```http
GET /ping
```

Response:

```text
pong
```

This endpoint checks whether the Go process is running. It does not query PostgreSQL.

### Readiness

```http
GET /readyz
```

Successful response:

```json
{
  "status": "ready"
}
```

This endpoint checks whether the application can reach PostgreSQL.

It returns `503 Service Unavailable` when the database is unavailable.

### Retrieve a Configuration

```http
GET /configs/{id}
```

Example:

```bash
curl http://localhost:8080/configs/cfg_1
```

Responses:

| Condition               | Status |
| ----------------------- | -----: |
| Configuration found     |  `200` |
| Invalid ID              |  `400` |
| Configuration not found |  `404` |
| Unexpected error        |  `500` |

### Create or Update a Configuration

```http
POST /configs
```

Example:

```bash
curl \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "id": "cfg_1",
    "host": "localhost",
    "port": 8080,
    "app_name": "config-service",
    "log_level": "INFO"
  }' \
  http://localhost:8080/configs
```

If the ID does not exist, the record is created.

If the ID already exists, the record is updated using PostgreSQL `ON CONFLICT DO UPDATE`.

Validation rules:

| Field       | Rule                                |
| ----------- | ----------------------------------- |
| `id`        | Required, maximum 64 characters     |
| `host`      | Required, maximum 255 characters    |
| `port`      | Between 1 and 65535                 |
| `app_name`  | Required, maximum 128 characters    |
| `log_level` | `DEBUG`, `INFO`, `WARN`, or `ERROR` |

## Local Access

Start port forwarding:

```bash
kubectl -n config-service port-forward service/config-service 8080:8080
```

The service is then available at:

```text
http://localhost:8080
```

## Database

PostgreSQL runs as a single-replica StatefulSet.

Its data is stored in a PersistentVolumeClaim named:

```text
postgres-data
```

The schema is defined in:

```text
migrations/001_create_configs.sql
```

The migration is executed by a Kubernetes Job.

Run it again with:

```bash
make migrate
```

The migration uses `CREATE TABLE IF NOT EXISTS`, so it can be safely repeated.

## Configuration and Secrets

Terraform creates a ConfigMap containing:

* `APP_PORT`
* `LOG_LEVEL`

Terraform creates a Kubernetes Secret containing:

* `DB_USER`
* `DB_PASSWORD`
* `DATABASE_URL`

The database password is generated using Terraform's random provider.

The password is not hardcoded in the application or Kubernetes manifests.

Terraform state may contain sensitive values and is excluded from Git.

For production, an external secret manager and encrypted remote Terraform state should be used.

## Health and Failure Behavior

Kubernetes uses:

```text
/ping   for liveness
/readyz for readiness
```

If PostgreSQL becomes unavailable:

* `/ping` continues to report that the process is alive;
* `/readyz` returns `503`;
* database-backed requests fail without exposing database credentials;
* Kubernetes stops routing traffic to the unready pod.

The application supports graceful shutdown when Kubernetes terminates the pod.

## Testing

Run Go tests:

```bash
make test
```

The tests cover:

* liveness;
* readiness;
* configuration creation;
* configuration retrieval;
* invalid input;
* missing configurations;
* service validation.

Run deployed-system smoke tests:

```bash
make validate
```

The smoke test verifies:

1. `/ping`;
2. `/readyz`;
3. configuration creation;
4. configuration retrieval;
5. configuration update;
6. missing configuration behavior.

## Useful Commands

```bash
make help
make test
make cluster-create
make image-build
make image-load
make infra-apply
make migrate
make deploy
make bootstrap
make validate
make status
make logs
make clean
```

## Troubleshooting

Show all resources:

```bash
make status
```

Show application logs:

```bash
make logs
```

Show PostgreSQL logs:

```bash
kubectl -n config-service logs postgres-0
```

Show migration logs:

```bash
kubectl -n config-service logs job/config-service-migration
```

Describe an application pod:

```bash
kubectl -n config-service describe pod -l app=config-service
```

Check Terraform state:

```bash
terraform -chdir=infra/terraform show
```

## Cleanup

Delete the local infrastructure and Kind cluster:

```bash
make clean
```

## Known Limitations

* PostgreSQL has one replica.
* There is no database high availability.
* There are no automated database backups.
* Storage is local to the Kind cluster.
* The application is exposed using port forwarding.
* TLS is not configured.
* Kubernetes Secrets are used instead of an external secret manager.
* Metrics and dashboards are not included.
