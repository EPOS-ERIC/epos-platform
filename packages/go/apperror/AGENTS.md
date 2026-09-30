# `apperror` Package Guidance

- Keep this package transport-neutral. Protocol mappings belong in adapter subpackages such as `grpc/` or in service HTTP boundaries.
- Treat `Code`, safe `Detail`, and `Target` as the cross-service contract. Causes are diagnostic only and must not be serialized.
- When changing an application outcome, update its safe default detail, HTTP and gRPC mappings, tests, and `README.md` together.
- Keep unknown outcomes and unclassified errors fail-closed as server errors.
- Preserve nested field targets and causes when prefixing or wrapping errors.
