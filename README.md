# EPOS Platform

Monorepo for EPOS platform services, shared packages, and local development tooling.

## Current status

The repository currently contains four active backend services used in local development:

- `services/metadata-lifecycle` (Kotlin): metadata lifecycle and versioning APIs backed by Neo4j.
- `services/gateway` (Go): API gateway/aggregation layer.
- `services/user-group` (Go): users, groups, memberships, and authorization-oriented data.
- `services/data-retriever` (Go): templated outbound retrieval, payload caching, and search.

Local infrastructure and service wiring for development are defined in `docker/compose.dev.yaml` and orchestrated by Tilt.

## Architecture (current dev setup)

```mermaid
flowchart TB
    %% Based on docker/compose.dev.yaml and Tilt image wiring

    DEV([Developer / Local Client])

    subgraph EDGE[Published host ports]
        GWE[gateway-service :32000]
        MLE[metadata-lifecycle-service :8080]
        UGE[user-group-service :8090]
        DRE[data-retriever-service :8100]
        N4J[neo4j :7474/7687]
        PUG[user-group-db :5432]
        PDR[data-retriever-db :5433]
    end

    subgraph NET["epos-network - docker compose"]
        GW[gateway-service]
        ML[metadata-lifecycle-service]
        UG[user-group-service]
        DR[data-retriever-service]

        NEO[(neo4j)]
        UGDB[(user-group-db)]
        DRDB[(data-retriever-db)]
    end

    subgraph SHARED[Shared Go packages used by dev images]
        LOG[packages/go/logging]
        PG[packages/go/postgres]
        MIG[packages/go/migrations]
    end

    DEV --> GWE
    DEV --> MLE
    DEV --> UGE
    DEV --> DRE
    DEV --> N4J

    GWE --- GW
    MLE --- ML
    UGE --- UG
    DRE --- DR
    N4J --- NEO
    PUG --- UGDB
    PDR --- DRDB

    GW -->|depends_on started| ML
    UG -->|depends_on healthy| UGDB
    DR -->|depends_on healthy| DRDB
    ML -->|depends_on healthy| NEO

    GW -. uses .-> LOG
    UG -. uses .-> LOG
    DR -. uses .-> LOG
    UG -. uses .-> PG
    DR -. uses .-> PG
    UG -. uses .-> MIG
    DR -. uses .-> MIG

    classDef svc fill:#0b5d8b,stroke:#083b59,color:#ffffff,stroke-width:1px;
    classDef infra fill:#175c4c,stroke:#0f3d32,color:#ffffff,stroke-width:1px;
    classDef db fill:#3f6b1d,stroke:#2d4c14,color:#ffffff,stroke-width:1px;
    classDef shared fill:#6f4b1f,stroke:#4f3516,color:#ffffff,stroke-width:1px;
    classDef edge fill:#39424e,stroke:#1f252d,color:#ffffff,stroke-width:1px;

    class GW,ML,UG,DR svc;
    class NEO infra;
    class UGDB,DRDB db;
    class LOG,PG,MIG shared;
    class GWE,MLE,UGE,DRE,N4J,PUG,PDR edge;
```

## Shared Go packages

Shared Go modules live in `packages/go/` and are reused across services.

- `packages/go/logging`: structured JSON logging with component loggers (`logging.Get("component")`).
- `packages/go/postgres`: shared PostgreSQL connection bootstrap from env configuration.
- `packages/go/migrations`: shared `.sql` migration runner (sorted file execution).

## Repository layout

```text
.
|- apps/
|- services/
|  |- gateway/
|  |- user-group/
|  |- data-retriever/
|  `- metadata-lifecycle/
|- packages/
|  |- go/
|  |- shacl/
|  `- scripts/
|- docker/
|- helm/
`- cli/
```

## Local development

From repository root:

```bash
make up
```

This starts Tilt and the local stack (service + infra containers).

Common local endpoints:

- Metadata Lifecycle API: `http://localhost:8080`
- Metadata Lifecycle Swagger: `http://localhost:8080/swagger`
- Gateway API: `http://localhost:32000`
- User Group API: `http://localhost:8090`
- Data Retriever API: `http://localhost:8100`
- Neo4j Browser: `http://localhost:7474`

Stop local development stack:

```bash
make down
```

## Repository commands

From repository root:

```bash
make up
make down
make test
make fmt
make lint
make populate
```

- `make test|fmt|lint` fan out to all current services.
- Service-specific build and Docker targets stay in each service `Makefile`.

## Service documentation

- Metadata Lifecycle: `services/metadata-lifecycle/README.md`
- Gateway: `services/gateway/README.md`
- User Group: `services/user-group/README.md`
- Data Retriever: `services/data-retriever/README.md`
