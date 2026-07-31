# Config Service Design

## 1. Overview

The system is a small Go application that manages configuration records.

It will:

* expose HTTP APIs;
* store data in PostgreSQL;
* run in a local Kind Kubernetes cluster;
* use Terraform and Kubernetes manifests for infrastructure;
* use a Makefile for repeatable setup, deployment, and validation.

---

## 2. Local Architecture

```text
Local user
    |
    | kubectl port-forward
    v
Application Service
    |
    v
Go Application
    |
    | SQL
    v
PostgreSQL Service
    |
    v
PostgreSQL
    |
    v
Persistent Volume
```

The Go application is deployed using a Kubernetes Deployment.

PostgreSQL is deployed using a StatefulSet with persistent storage.

---

## 3. API Contract

### `GET /ping`

Checks whether the application process is running.

Successful response:

```text
pong
```

Status:

```text
200 OK
```

---

### `GET /readyz`

Checks whether the application can connect to PostgreSQL.

Possible responses:

| Condition                 | Status |
| ------------------------- | -----: |
| PostgreSQL is reachable   |  `200` |
| PostgreSQL is unavailable |  `503` |

---

### `GET /configs/{id}`

Returns the configuration with the given ID.

The ID is a unique string that identifies a configuration record.

Successful response:

```json
{
  "id": "cfg_1",
  "host": "localhost",
  "port": 8080,
  "app_name": "config-service",
  "log_level": "INFO"
}
```

Possible responses:

| Condition        | Status |
| ---------------- | -----: |
| Config found     |  `200` |
| Invalid ID       |  `400` |
| Config not found |  `404` |
| Database error   |  `503` |

---

### `POST /configs`

Creates or updates a configuration.

Example request:

```json
{
  "id": "cfg_1",
  "host": "localhost",
  "port": 8080,
  "app_name": "config-service",
  "log_level": "INFO"
}
```

If the ID does not exist, a new record is created.

If the ID already exists, the record is updated.

PostgreSQL will perform the upsert using:

```sql
INSERT ... ON CONFLICT DO UPDATE
```

---

## 4. Application Structure

```text
HTTP Handler
    |
    v
Service
    |
    v
Repository
    |
    v
PostgreSQL
```

### Handler

Responsible for:

* HTTP routes;
* request decoding;
* response encoding;
* HTTP status codes.

### Service

Responsible for:

* validation;
* application rules;
* calling the repository.

### Repository

Responsible for:

* PostgreSQL queries;
* creating, updating, and retrieving records;
* translating database errors.

---

## 5. Validation

| Field       | Validation          |
| ----------- | ------------------- |
| `id`        | Required            |
| `host`      | Required            |
| `port`      | Between 1 and 65535 |
| `app_name`  | Required            |
| `log_level` | Required            |

Invalid requests return:

```text
400 Bad Request
```

Database errors are not returned directly to clients.

---

## 6. Database

The database contains a `configs` table.

```sql
CREATE TABLE configs (
    id VARCHAR(64) PRIMARY KEY,
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
    app_name VARCHAR(128) NOT NULL,
    log_level VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

The `id` is the primary key.

No additional index is required because records are retrieved using the primary key.

The schema is created using a database migration executed through a Kubernetes Job.

---

## 7. Infrastructure Components

The local environment contains:

* Kind Kubernetes cluster;
* Kubernetes namespace;
* Go application Deployment;
* application Service;
* PostgreSQL StatefulSet;
* PostgreSQL Service;
* PersistentVolumeClaim;
* ConfigMap;
* Secret;
* migration Job;
* liveness and readiness probes.

Terraform manages the local infrastructure resources.

Kubernetes manifests manage the application deployment and migration Job.

---

## 8. Configuration and Secrets

Non-sensitive configuration is stored in a Kubernetes ConfigMap.

Examples:

* application port;
* database host;
* database port;
* database name.

Sensitive configuration is stored in a Kubernetes Secret.

Examples:

* database username;
* database password.

The application receives these values through environment variables.

For a production environment, an external secret-management system would be preferred.

---

## 9. Health Checks

### Liveness

The liveness probe calls:

```text
GET /ping
```

It checks whether the Go process is running.

It does not depend on PostgreSQL.

### Readiness

The readiness probe calls:

```text
GET /readyz
```

It checks whether PostgreSQL is reachable.

When PostgreSQL is unavailable, the application becomes unready and stops receiving traffic.

---

## 10. Persistence

PostgreSQL uses a PersistentVolumeClaim.

This allows database data to survive a PostgreSQL container restart.

The Go application does not store configuration data inside its container.

---

## 11. Deployment Flow

```text
Create Kind cluster
    |
Build Docker image
    |
Load image into Kind
    |
Run Terraform
    |
Wait for PostgreSQL
    |
Run database migration
    |
Deploy Go application
    |
Wait for rollout
    |
Run validation
```

The workflow is automated using Makefile targets and scripts.

The main setup command will be:

```bash
make bootstrap
```

---

## 12. Validation

The deployment validation will check that:

* the Kubernetes cluster is reachable;
* PostgreSQL is ready;
* the migration completes;
* the application rollout completes;
* `/ping` returns `pong`;
* `/readyz` reports that the application is ready;
* a configuration can be created;
* the configuration can be retrieved;
* an existing configuration can be updated.

---

## 13. Observability

The application writes logs to standard output.

Logs include:

* application startup;
* request method and path;
* response status;
* request duration;
* database failures.

Database passwords and other secret values are not logged.

Kubernetes liveness and readiness probes provide basic health visibility.

---

## 14. Operational Assumptions

* Docker is running locally.
* Kind, Docker, Go, Terraform, `kubectl`, and Make are installed.
* The environment is intended for local use.
* The application is accessed using `kubectl port-forward`.
* One PostgreSQL instance is sufficient for the assignment.

---

## 15. Known Limitations

* PostgreSQL runs as a single replica.
* There is no database high availability.
* Storage is local to the Kind cluster.
* There are no automated database backups.
* There is no Ingress controller.
* There is no TLS.
* Secrets are stored using Kubernetes Secrets.
* There is no complete metrics or dashboard setup.
