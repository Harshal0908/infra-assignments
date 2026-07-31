# Config Service Design

## 1. Goal

The goal of this project is to build a small configuration management service.

The service will:

* be written in Go;
* expose HTTP APIs;
* store configuration records in PostgreSQL;
* run inside a local Kubernetes cluster;
* use Terraform for infrastructure provisioning;
* provide repeatable setup and deployment commands;
* include health checks, logs, tests, and troubleshooting instructions.

---

## 2. Technology Choices

The project will use the following tools:

| Tool                 | Purpose                               |
| -------------------- | ------------------------------------- |
| Go                   | Build the HTTP application            |
| PostgreSQL           | Store configuration records           |
| Docker               | Package the Go application            |
| Kind                 | Create a local Kubernetes cluster     |
| Kubernetes           | Run the application and database      |
| Terraform            | Provision infrastructure resources    |
| Kubernetes manifests | Deploy application resources          |
| Makefile             | Provide simple repeatable commands    |
| GitHub Actions       | Run automated checks, if time permits |

### Why Kind?

Kind creates a Kubernetes cluster using Docker containers.

It was chosen because:

* it is lightweight;
* it works well on a local computer;
* it is easy to create and delete;
* application images can be loaded directly into the cluster;
* it does not require a cloud account.

### Why PostgreSQL?

It provides:

* reliable persistent storage;
* database constraints;
* fast primary-key lookups;
* safe atomic upsert operations.

### Why Terraform?

Terraform provides a repeatable way to create infrastructure.

Instead of manually creating every resource, Terraform describes the desired infrastructure in code.

Running Terraform again should safely move the system toward the same desired state.

### Why Kubernetes manifests?

Kubernetes manifests clearly describe how the application should run.

They are a natural fit for:

* Deployments;
* Services;
* health probes;
* migration Jobs;
* resource configuration.

---

## 3. High-Level Architecture

```text
Developer laptop
       |
       | make bootstrap
       v
+-------------------------------------+
| Kind Kubernetes Cluster             |
|                                     |
|  +-------------------------------+  |
|  | Namespace: config-service     |  |
|  |                               |  |
|  |  +-------------------------+  |  |
|  |  | Kubernetes Service      |  |  |
|  |  | config-service          |  |  |
|  |  +------------+------------+  |  |
|  |               |               |  |
|  |               v               |  |
|  |  +-------------------------+  |  |
|  |  | Go Application          |  |  |
|  |  | Deployment              |  |  |
|  |  |                         |  |  |
|  |  | GET /ping               |  |  |
|  |  | GET /readyz             |  |  |
|  |  | GET /configs/{id}       |  |  |
|  |  | POST /configs           |  |  |
|  |  +------------+------------+  |  |
|  |               | SQL           |  |
|  |               v               |  |
|  |  +-------------------------+  |  |
|  |  | PostgreSQL Service      |  |  |
|  |  +------------+------------+  |  |
|  |               |               |  |
|  |               v               |  |
|  |  +-------------------------+  |  |
|  |  | PostgreSQL StatefulSet  |  |  |
|  |  +------------+------------+  |  |
|  |               |               |  |
|  |               v               |  |
|  |  +-------------------------+  |  |
|  |  | Persistent Volume Claim |  |  |
|  |  +-------------------------+  |  |
|  |                               |  |
|  |  Migration Job ------------->|  |
|  |  PostgreSQL                   |  |
|  +-------------------------------+  |
+-------------------------------------+
       ^
       |
       | kubectl port-forward
       |
Local API client
```

---

## 4. Main Components

### 4.1 Go Application

The Go application will expose the HTTP APIs.

It will:

* receive HTTP requests;
* validate request data;
* call the service layer;
* store and retrieve records through the repository layer;
* return clear HTTP responses;
* expose health-check endpoints;
* write structured logs;
* shut down gracefully.

### 4.2 PostgreSQL

PostgreSQL will store all configuration records.

The database will run inside Kubernetes using a StatefulSet.

A PersistentVolumeClaim will store the database files.

This means the database data should remain available when the PostgreSQL container restarts.

### 4.3 Migration Job

A Kubernetes Job will create the database schema.

The migration will run after PostgreSQL is ready and before the application is considered fully deployed.

The application will not create tables directly during normal startup.

This keeps database changes separate and visible.

### 4.4 Kubernetes Service

The application Service will provide a stable internal address for the Go application.

For local testing, the user will access the service using:

```bash
kubectl port-forward
```

Port forwarding was chosen because it is simple and does not require installing an Ingress controller.

### 4.5 ConfigMap

The ConfigMap will contain non-sensitive settings, such as:

* application port;
* database host;
* database port;
* database name;
* log level.

### 4.6 Secret

The Kubernetes Secret will contain sensitive values, such as:

* database username;
* database password.

Passwords will not be hardcoded inside the Go application.

---

## 5. Application Structure

The Go application will use separate layers.

```text
HTTP Request
     |
     v
Handler
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

### Handler Layer

The handler layer understands HTTP.

It will:

* read URL parameters;
* decode JSON request bodies;
* call the service layer;
* choose the correct HTTP status code;
* encode JSON responses.

The handler will not contain SQL queries.

### Service Layer

The service layer contains the application rules.

It will:

* validate configuration values;
* decide how operations should behave;
* call the repository;
* return known application errors.

### Repository Layer

The repository layer understands PostgreSQL.

It will:

* execute SQL queries;
* insert or update records;
* retrieve records;
* translate database errors into known repository errors.

The repository will not write HTTP responses.

### Domain Layer

The domain layer contains shared application data types.

Example:

```go
type Config struct {
    ID        string    `json:"id"`
    Host      string    `json:"host"`
    Port      int       `json:"port"`
    AppName   string    `json:"app_name"`
    LogLevel  string    `json:"log_level"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

---

## 6. API Design

## 6.1 Liveness Endpoint

```text
GET /ping
```

Successful response:

```text
pong
```

HTTP status:

```text
200 OK
```

### Purpose

This endpoint answers:

> Is the Go application process alive?

It will not query PostgreSQL.

A database failure should not cause Kubernetes to restart a healthy Go process repeatedly.

---

## 6.2 Readiness Endpoint

```text
GET /readyz
```

### Purpose

This endpoint answers:

> Is the application ready to handle database-backed requests?

The endpoint will check the PostgreSQL connection.

When PostgreSQL is reachable:

```json
{
  "status": "ready"
}
```

HTTP status:

```text
200 OK
```

When PostgreSQL is unavailable:

```json
{
  "status": "not_ready"
}
```

HTTP status:

```text
503 Service Unavailable
```

---

## 6.3 Retrieve Configuration

```text
GET /configs/{id}
```

Example:

```text
GET /configs/cfg_1
```

Successful response:

```json
{
  "id": "cfg_1",
  "host": "localhost",
  "port": 8080,
  "app_name": "config-service",
  "log_level": "INFO",
  "created_at": "2026-07-31T10:00:00Z",
  "updated_at": "2026-07-31T10:00:00Z"
}
```

HTTP status:

```text
200 OK
```

When the configuration does not exist:

```json
{
  "error": {
    "code": "config_not_found",
    "message": "configuration was not found"
  }
}
```

HTTP status:

```text
404 Not Found
```

When the ID is invalid:

```json
{
  "error": {
    "code": "invalid_id",
    "message": "configuration ID is invalid"
  }
}
```

HTTP status:

```text
400 Bad Request
```

---

## 6.4 Create or Update Configuration

```text
POST /configs
```

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

The operation behaves as an upsert.

This means:

* if the ID does not exist, a new record is created;
* if the ID already exists, the existing record is updated.

The operation will use PostgreSQL:

```sql
INSERT ... ON CONFLICT DO UPDATE
```

This allows PostgreSQL to perform the operation safely as one database action.

Successful response:

```json
{
  "id": "cfg_1",
  "host": "localhost",
  "port": 8080,
  "app_name": "config-service",
  "log_level": "INFO",
  "created_at": "2026-07-31T10:00:00Z",
  "updated_at": "2026-07-31T10:00:00Z"
}
```

HTTP status:

```text
200 OK
```

---

## 7. Validation Rules

The application will validate incoming data before saving it.

| Field       | Rule                                |
| ----------- | ----------------------------------- |
| `id`        | Required, maximum 64 characters     |
| `host`      | Required, maximum 255 characters    |
| `port`      | Must be between 1 and 65535         |
| `app_name`  | Required, maximum 128 characters    |
| `log_level` | Must be DEBUG, INFO, WARN, or ERROR |

Invalid requests will return:

```text
400 Bad Request
```

Database failures will not expose raw database messages to the client.

Unexpected database failures will return:

```text
503 Service Unavailable
```

or:

```text
500 Internal Server Error
```

depending on the failure.

---

## 8. Database Design

The database will contain a `configs` table.

```sql
CREATE TABLE configs (
    id VARCHAR(64) PRIMARY KEY,
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
    app_name VARCHAR(128) NOT NULL,
    log_level VARCHAR(10) NOT NULL
        CHECK (log_level IN ('DEBUG', 'INFO', 'WARN', 'ERROR')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Why use `id` as the primary key?

Each configuration needs a unique identifier.

The primary key:

* prevents duplicate IDs;
* creates an index automatically;
* makes lookups by ID fast;
* supports PostgreSQL upsert operations.

### Why are there no additional indexes?

The current API only retrieves records using the primary key.

The primary-key index is enough for this use case.

Extra indexes would add complexity and slow down writes without helping the current API.

### Why use database constraints?

Application validation can contain bugs.

Database constraints provide a second safety layer.

For example, PostgreSQL will reject:

* invalid port numbers;
* empty required fields;
* unsupported log levels.

---

## 9. Infrastructure Responsibility Split

The project will use a clear split between tools.

### Kind

Kind will create and delete the local Kubernetes cluster.

The cluster lifecycle will be controlled through scripts or Makefile commands.

### Terraform

Terraform will manage foundational Kubernetes infrastructure.

The planned Terraform-managed resources include:

* namespace;
* configuration values;
* database credentials;
* PostgreSQL Service;
* PostgreSQL StatefulSet;
* persistent storage.

### Kubernetes Manifests

Kubernetes manifests will manage application deployment resources.

The planned manifest-managed resources include:

* migration Job;
* Go application Deployment;
* application Service;
* liveness probe;
* readiness probe.

### Makefile

The Makefile will connect all steps in the correct order.

This split keeps each tool focused on work it handles clearly.

---

## 10. Deployment Flow

The complete deployment flow will be:

```text
1. Check required tools
        |
        v
2. Create Kind cluster
        |
        v
3. Build Go application image
        |
        v
4. Load image into Kind
        |
        v
5. Run Terraform
        |
        v
6. Start PostgreSQL
        |
        v
7. Wait for PostgreSQL readiness
        |
        v
8. Run database migration Job
        |
        v
9. Deploy Go application
        |
        v
10. Wait for application rollout
        |
        v
11. Run smoke tests
```

The main command will be similar to:

```bash
make bootstrap
```

The goal is for a reviewer to avoid manually remembering the deployment order.

---

## 11. Planned Makefile Commands

The Makefile will provide simple commands.

```text
make help
make cluster-create
make cluster-delete
make image-build
make image-load
make infra-init
make infra-plan
make infra-apply
make migrate
make deploy
make validate
make logs
make status
make clean
make bootstrap
```

### Main Commands

`make bootstrap`

Creates and deploys the complete environment.

`make validate`

Runs health checks and API smoke tests.

`make clean`

Deletes the local environment.

`make logs`

Shows application logs.

`make status`

Shows Kubernetes resource status.

---

## 12. Health-Check Design

### Liveness Probe

The Kubernetes liveness probe will call:

```text
GET /ping
```

If this endpoint repeatedly fails, Kubernetes can restart the application container.

### Readiness Probe

The Kubernetes readiness probe will call:

```text
GET /readyz
```

If PostgreSQL is unavailable, the application pod will become unready.

Kubernetes will stop sending normal traffic to that pod.

The pod does not need to restart simply because the database is temporarily unavailable.

---

## 13. Database Failure Behavior

### Database unavailable during startup

The application will try to create its PostgreSQL connection pool.

If the database cannot be reached:

* the application will log a clear error;
* the application will fail startup;
* Kubernetes will restart it using its normal restart behavior.

The database password will never appear in logs.

### Database unavailable while running

If PostgreSQL becomes unavailable:

* `/ping` will still return `200`;
* `/readyz` will return `503`;
* database-backed endpoints will return a controlled error;
* the failure will be written to the application logs.

---

## 14. Logging and Observability

The application will write logs to standard output.

Kubernetes will collect these container logs automatically.

The logs should include:

* log level;
* message;
* HTTP method;
* request path;
* response status;
* request duration;
* request ID;
* database error category when relevant.

Example:

```json
{
  "level": "info",
  "message": "request completed",
  "method": "GET",
  "path": "/configs/cfg_1",
  "status": 200,
  "duration_ms": 4,
  "request_id": "request-123"
}
```

Startup logs should show:

* application start;
* listening port;
* database host;
* database name;
* readiness state.

Startup logs must not show:

* database password;
* complete database connection string containing credentials;
* secret values.

Prometheus metrics may be added if the required features are complete first.

---

## 15. Graceful Shutdown

Kubernetes sends a termination signal when stopping a pod.

The Go application will listen for this signal.

During shutdown, it will:

1. stop accepting new requests;
2. allow active requests a short time to finish;
3. close the database connection pool;
4. exit cleanly.

This reduces interrupted requests during application restarts and deployments.

---

## 16. Testing Strategy

The project will include several kinds of validation.

### Unit Tests

Unit tests will cover:

* `/ping`;
* valid configuration upsert;
* missing configuration ID;
* invalid JSON;
* invalid port;
* invalid log level;
* successful retrieval;
* configuration not found;
* repository failure handling.

### Repository Tests

Repository tests will verify:

* inserting a new record;
* updating an existing record;
* retrieving a record;
* returning a known not-found error.

### Deployment Validation

The deployment process will verify:

* Kubernetes cluster is reachable;
* PostgreSQL pod is ready;
* migration Job succeeds;
* application Deployment becomes ready;
* application Service exists.

### Smoke Tests

The smoke-test flow will:

1. open a local port-forward;
2. call `/ping`;
3. call `/readyz`;
4. create a configuration;
5. retrieve the configuration;
6. update the configuration;
7. retrieve the updated configuration;
8. check the expected values;
9. stop the port-forward.

---

## 17. Security Approach

For the local assignment:

* database credentials will be stored in a Kubernetes Secret;
* credentials will not be committed as plain text;
* logs will not print secret values;
* the application will use environment variables for configuration;
* SQL queries will use parameters instead of joining user input into SQL strings.

Parameterized SQL helps prevent SQL injection.

### Production Improvements

For a production system, possible improvements include:

* cloud secret manager or HashiCorp Vault;
* External Secrets Operator;
* encrypted remote Terraform state;
* database password rotation;
* TLS between services;
* Kubernetes network policies;
* restricted service accounts;
* container image scanning;
* non-root containers;
* read-only container file systems.

These production tools are not required for the local assignment.

---

## 18. Local Networking

The Go application will communicate with PostgreSQL using the PostgreSQL Kubernetes Service name.

Example:

```text
postgres.config-service.svc.cluster.local
```

Users outside the cluster will access the application with:

```bash
kubectl port-forward service/config-service 8080:8080
```

The local API will then be available at:

```text
http://localhost:8080
```

An Ingress controller will not be installed because port forwarding is enough for local validation.

---

## 19. Repository Structure

The planned repository structure is:

```text
submission/harshal-humne/
├── cmd/
│   └── main.go
├── docs/
│   └── design.md
├── internal/
│   ├── config/
│   ├── domain/
│   ├── handler/
│   ├── repository/
│   │   └── postgres/
│   └── service/
├── migrations/
│   └── 001_create_configs.sql
├── infra/
│   └── terraform/
├── k8s/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── migration-job.yaml
├── scripts/
│   ├── create-cluster.sh
│   └── smoke-test.sh
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

This layout keeps application code, infrastructure code, deployment files, documentation, and automation separate.

---

## 20. Repeatability

The setup should be safe to run more than once.

Examples:

* Kind cluster creation will check whether the cluster already exists;
* Terraform will compare the current state with the desired state;
* Kubernetes manifests will update existing resources;
* database migrations will avoid recreating an existing table;
* PostgreSQL upsert will safely update an existing record.

The goal is to avoid requiring manual cleanup between normal deployment attempts.

---

## 21. Known Limitations

The first version intentionally has the following limitations:

* one local Kubernetes cluster;
* one PostgreSQL replica;
* no PostgreSQL high availability;
* local persistent storage only;
* no automatic database backups;
* no TLS;
* no Ingress controller;
* no external secret manager;
* no automatic secret rotation;
* no autoscaling;
* no complete monitoring stack;
* no multi-environment setup;
* no production container registry.

These limitations keep the assignment small, understandable, and achievable.

---

## 22. Possible Production Improvements

For production, the system could be improved by adding:

* managed PostgreSQL;
* automated backups;
* PostgreSQL high availability;
* external secret management;
* TLS;
* network policies;
* resource requests and limits based on measurements;
* autoscaling;
* Prometheus metrics;
* Grafana dashboards;
* distributed tracing;
* centralized logging;
* vulnerability scanning;
* automated image publishing;
* GitOps deployment;
* separate development, staging, and production environments.

These are documented improvements, not requirements for the local assignment.

---

## 23. Main Design Decisions

The main design decisions are:

1. Use Kind because it provides a simple local Kubernetes environment.
2. Use PostgreSQL because persistent database storage is required.
3. Use a StatefulSet and persistent volume for the database.
4. Use a Deployment for the stateless Go application.
5. Keep liveness separate from database readiness.
6. Use a migration Job instead of changing the schema during application startup.
7. Use ConfigMaps for non-secret values.
8. Use Secrets for database credentials.
9. Use port forwarding instead of installing an Ingress controller.
10. Use Makefile commands to make the workflow repeatable.
11. Prioritize a small reliable solution over unnecessary infrastructure.

---

## 24. Success Criteria

The project is considered successful when:

* a developer can create the cluster using documented commands;
* PostgreSQL starts successfully;
* the database migration runs successfully;
* the Go application becomes ready;
* `/ping` returns `pong`;
* `/readyz` confirms database connectivity;
* configurations can be created;
* existing configurations can be updated;
* configurations can be retrieved by ID;
* invalid requests return clear errors;
* missing configurations return `404`;
* the application writes useful logs;
* database data survives an application restart;
* the complete deployment can be repeated;
* automated smoke tests pass;
* cleanup can be performed using one documented command.
