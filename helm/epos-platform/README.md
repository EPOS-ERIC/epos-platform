# EPOS Platform Helm Chart

This chart deploys the current EPOS platform backend services:

- `gateway`
- `metadata-lifecycle`
- `user-group`
- `data-retriever`

It also includes optional in-chart PostgreSQL and Neo4j instances for development-style deployments. For production, set the relevant datastore `enabled` flags to `false` and provide external connection settings in `values.yaml` or an environment-specific values file.

## Validate

```bash
helm lint helm/epos-platform
helm template epos-platform helm/epos-platform
```
