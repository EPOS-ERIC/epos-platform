# AGENTS

## Service Conventions

- Keep business logic in domain-oriented packages.
- Keep transport layer thin and validation explicit.
- The `configuration` payload is schemaless JSON: validate representation only (valid JSON, size limits), never its shape.
- PUT only, no PATCH. Required field is `T`, nullable scalar is `*T`, collection is `[]T`; absent and `null` both decode to `nil` and mean clear. `is_public` is an exception: absent means public.
- Ownership comes from gRPC metadata (`x-epos-user-id`, `x-epos-admin`), never from the request body. The gateway is the only client and attaches them after bearer validation.
- Enforce visibility and ownership in `grpcapi/`, so policy holds regardless of the caller: anonymous sees public only; private is owner-or-admin; official implies public and requires admin to set or change.
