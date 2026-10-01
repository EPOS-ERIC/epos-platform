# `apperror` Package Guidance

- This package owns shared application codes, safe default details, and their HTTP status mapping. Keep HTTP response encoding in service boundaries; protocol-specific mappings such as gRPC status and details belong in adapter subpackages such as `grpc/`.
- Treat `Code`, safe `Detail`, and `Target` as the cross-service contract. Causes are diagnostic only and must not be serialized.
- When changing an application outcome, update its safe default detail, HTTP and gRPC mappings, tests, and `README.md` together.
- Keep unknown outcomes and unclassified errors fail-closed as server errors.
- Preserve nested field targets and causes when prefixing or wrapping errors.
