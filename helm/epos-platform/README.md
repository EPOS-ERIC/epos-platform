# EPOS Platform Helm Chart

This chart deploys the current EPOS platform apps and backend services:

- `platform`
- `gateway`
- `metadata-lifecycle`
- `user-group`
- `data-retriever`
- `oss-aai`

It also includes optional in-chart PostgreSQL and Neo4j instances for development-style deployments. For production, set the relevant datastore `enabled` flags to `false` and provide external connection settings in `values.yaml` or an environment-specific values file.

The `platform` frontend expects the gateway API at `/v1`. When `platform.ingress.enabled=true`, the chart routes `/v1` to the gateway service and `/` to the frontend service on the same host.

## Validate

```bash
helm lint helm/epos-platform
helm template epos-platform helm/epos-platform
```

Equivalent root Make targets are available:

```bash
make helm-lint
make helm-template
```

## Deploy

Deploy or update the chart with:

```bash
make helm-deploy
```

The Make targets support the same Kubernetes context variable used by the Tilt targets:

```bash
make up KUBE_CONTEXT=kind-epos
make helm-deploy KUBE_CONTEXT=kind-epos HELM_NAMESPACE=epos-platform
make helm-status KUBE_CONTEXT=kind-epos HELM_NAMESPACE=epos-platform
make helm-uninstall KUBE_CONTEXT=kind-epos HELM_NAMESPACE=epos-platform
```

Override chart values with `HELM_VALUES` or additional Helm arguments with `HELM_SET_ARGS`:

```bash
make helm-deploy HELM_VALUES=helm/epos-platform/values.yaml HELM_SET_ARGS='--set gateway.image.tag=dev'
```
