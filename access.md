# Distribution Access Specification

**Status:** agreed requirements for initial implementation, with future conversion requirements and deferred work identified below.

## 1. Purpose and scope

Provide distribution-oriented access without requiring clients to construct outbound HTTP requests.

For web-service access, clients supply:

- Distribution identifier.
- Parameter values.
- Optional native format selection or configured conversion selection.

Backend resolves metadata, authorizes access, prepares outbound request, executes it and streams response.

Initial implementation supports:

- Web-service access through a parameterized endpoint.
- Downloadable-file retrieval through a separate endpoint.
- Native format selection for web services.
- Structured failure responses.

Converter does not exist yet. Conversion selection and discovery contracts are defined for web-service distributions only. Downloadable-file distributions are never converted.

---

## 2. Public access API

### Endpoint

```http
POST /distributions/{distributionId}/access
```

This endpoint is for web-service distributions. Downloadable-file distributions use `GET /distributions/{distributionId}/download`.

`distributionId` is a UUID.

### Query parameters

| Parameter       | Required | Meaning                                                    |
| --------------- | -------: | ---------------------------------------------------------- |
| `output_format` |       No | Native format selector, scoped to distribution.            |
| `conversion_id` |       No | Stored plugin-distribution conversion relation identifier. |

`conversion_id` is a UUID; `output_format` is a non-empty string. Each query parameter may occur at most once; empty or repeated selectors are invalid input.

This is the complete target OpenAPI query contract. Initial implementation supports native formats only; add a TODO to connect `conversion_id` when converter and conversion-relation support are implemented. Do not build plugin registries, relation persistence or conversion clients as part of initial access.

### Selection rules

1. Resolve the specific distribution and its owning dataset version by distribution ID.
2. Distribution IDs are globally unique and version-owned; no separate version selector is needed.
3. Published dataset versions allow anonymous access. Every other state requires an authenticated actor with access under existing dataset permissions.
4. Unknown or deleted distribution IDs return 404. Never fall back to another version or remap stale IDs.
5. `output_format` and `conversion_id` are mutually exclusive; supplying both fails validation.
6. Without either format selector, use native default output without conversion.
7. `conversion_id` determines plugin, source format and output format; client does not repeat those values.
8. Format selectors match declared values exactly after normal URL decoding. No case folding, MIME normalization or aliases.

### Examples

```http
# Published distribution, native default
POST /distributions/{distributionId}/access

# Published distribution, selected native format
POST /distributions/{distributionId}/access?output_format=xml_dc

# Published distribution, selected conversion
POST /distributions/{distributionId}/access?conversion_id={conversionId}

```

Distribution identity does not freeze upstream data. Draft metadata may also remain mutable.

### Request body

JSON object and `parameters` array are required:

```json
{
  "parameters": [
    {
      "id": "9cf1103f-2e90-4742-a64a-d973e7d32e7d",
      "value": "mystation"
    }
  ]
}
```

When no caller values are needed:

```json
{
  "parameters": []
}
```

Missing body or missing `parameters` is invalid.

Public request does not accept arbitrary:

- URL or URL template.
- HTTP method.
- Headers.
- Rendered request body.
- Plugin ID.
- Converter input format.
- Execution mode.
- Dedicated upstream bearer-token field.

The request object and each parameter entry reject unknown fields. Each entry has exactly `id` and `value`. `id` is a UUID. `value` is a non-null JSON scalar or array; arrays are allowed only for multi-value parameters. Objects are not valid values.

Multi-value parameters accept an array:

```json
{
  "parameters": [
    {
      "id": "9cf1103f-2e90-4742-a64a-d973e7d32e7d",
      "value": ["myvalue1", "myvalue2"]
    }
  ]
}
```

---

## 3. Metadata identity and authorization

### Identifier lifecycle

Keep existing metadata-instance identifiers.

- Distribution and parameter IDs are those returned by selected metadata.
- Stability across edits or new versions is not guaranteed.
- Clients reload metadata after edits/version changes.
- Stale IDs fail; backend must not remap by name.
- Conversion relations target concrete distribution instances.
- Conversion relation must belong to selected distribution and apply to selected metadata instance.

No cross-version identity preservation or automatic conversion-relation remapping is introduced.

### Authentication

Platform authentication follows existing visibility rules:

- Distributions owned by published dataset versions permit anonymous access.
- Distributions in any other state require an authenticated actor with access to the owning dataset under existing permissions.
- Inaccessible metadata must not be disclosed through errors.

Platform authentication credentials must never be forwarded upstream automatically.

---

## 4. Parameter input requirements

### Unified addressing

One `parameters` list addresses:

- Path parameters.
- Query parameters.
- Header parameters.
- Request-body parameters.
- Scalar children of template parameters.

All caller-addressable parameters require IDs.

Client does not supply `kind`, location or separate `body_parameters`. Metadata determines type, placement and template membership.

### Value validation

Before outbound execution:

- Reject unknown IDs.
- Reject duplicate IDs.
- Reject IDs outside selected distribution metadata.
- Reject wrong JSON types.
- Enforce metadata constraints, including allowed values, patterns and ranges.
- Reject explicit `null`.
- Reject empty arrays.
- Reject direct caller assignments to backend-managed (read only) parameters.
- Apply declared defaults to omitted inputs.
- Reject missing required values without defaults.

Scalar inputs use matching JSON types. Multi-value inputs use non-empty arrays of valid values.

### Datetime values

Client sends RFC 3339 strings with explicit timezone:

```json
{
  "id": "parameter-uuid",
  "value": "2026-10-12T09:30:00Z"
}
```

Metadata Lifecycle:

1. Parses instant.
2. Validates configured datetime range.
3. Formats value using upstream metadata layout.

Client does not construct upstream-specific datetime strings.

### Backend-managed parameters

`read_only=true` means hidden/backend-managed, not immutable.

- Client must not supply such parameter through normal access inputs.
- Backend may assign it during request preparation.
- Format selection assigns encoding-format parameter through `output_format` or conversion relation.

### Template parameters and request-body templates

All templates share the same value-resolution and activation rules. Backend applies defaults, validates child values and renders templates; clients supply scalar child values by parameter ID.

The POST access JSON body and its `parameters` array are always required. Separately, add `body_required BOOLEAN NOT NULL DEFAULT TRUE` to `web_service_bodies`. The system has not been deployed, so update the original migration in place; no follow-up migration or existing-row backfill is needed.

Propagate `body_required` through web-service metadata CRUD: POST/PUT inputs accept it alongside configured request-body metadata (`body_content_type`, `body_template` and `body_parameters`); omitted values and imports default to `true`; GET responses return it alongside configured body metadata. If no body is configured, GET omits the field and POST/PUT reject a supplied `body_required`. DELETE endpoints need no field; normal deletion of body metadata removes its row and value.

**Required templates (`required=true` TEMPLATE parameter or `body_required=true`):**

- Always evaluated.
- Every referenced child must resolve from client input or a metadata default; otherwise return 400.

**Optional templates (`required=false` TEMPLATE parameter or `body_required=false`):**

- Render only when every declared child resolves from client input or a metadata default, even without caller-supplied values.
- If unresolved and client supplied no children, omit the optional template parameter or request body.
- If client supplied any child but another child remains unresolved, return 400; never silently discard supplied input.
- A request-body template with no child parameters is fully resolved and is sent even when `body_required=false`.

An optional request body follows the same rules as an optional TEMPLATE parameter. If omitted, the outbound HTTP method remains unchanged.

Request templates use existing Go `text/template` behavior and interpolate values as-is. No JSON escaping helper is provided initially; quotes or backslashes in inserted values may make a JSON request body invalid. TODO: add escaping only if real use requires it.

### Request value serialization

- Apply metadata defaults before serializing values.
- An omitted optional scalar without a default is absent from the outbound request.
- STRING values are used as-is; BOOLEAN uses `true` or `false`; INTEGER is signed 64-bit decimal; FLOAT is a finite JSON number using standard decimal formatting.
- DATETIME is normalized to UTC, then formatted using its configured layout; without a layout, use RFC 3339 UTC.
- Client supplies arrays for multi-value parameters; each item must match the declared scalar type, and arrays must be non-empty.
- QUERY with `explode=true` emits repeated keys (`key=a&key=b`). QUERY with `explode=false`, PATH and HEADER emit comma-separated values.
- Escape each PATH/QUERY value for its URL component before joining values; keep the comma separator unescaped. Do not URL-encode HEADER values; reject header values containing CR or LF.
- Do not coerce strings into numbers or booleans.
- No Content-Type is inferred from format selectors or request-body metadata. An outbound `Content-Type` is sent only when declared as a HEADER parameter.

---

## 5. Native format metadata

### Format-control property

Only exact property value is recognized:

```text
schema:encodingFormat
```

### Controller constraints

A web service may contain **at most one** format-control parameter across:

- Top-level PATH, QUERY or HEADER parameters.

Format control is not supported on TEMPLATE parameters, request-body parameters or nested template children. Do not add `property` to scalar parameter metadata. TODO: add property field only if real use requires it.

Format-control parameter must:

- Be STRING.
- Be single-value.
- Have `read_only=true`.
- Have valid default.
- Use one of the following definitions.

| Definition         | Requirements                                           |
| ------------------ | ------------------------------------------------------ |
| Fixed format       | Default present; no enum.                              |
| Selectable formats | Non-empty enum; default present and belonging to enum. |

Multiple controllers, a TEMPLATE controller, other types, multi-value controllers, unrestricted definitions and invalid defaults are metadata validation errors.

### Native format derivation

When controller exists:

- Fixed definition produces one native option.
- Enum definition produces declared native options.
- Default identifies native default selection.
- `web_service.format` is not added as another option.

When no controller exists:

- `web_service.format` is sole native/default option.
- Explicit matching `output_format` is accepted without changing outbound request.
- Other selectors fail validation.

Format values are opaque service selectors, such as `geojson` or `xml_dc`. They need not be MIME types.

---

## 6. Conversion relations

Future stored plugin-to-web-service-distribution relation declares:

```text
id
plugin_id
distribution_id
input_format
output_format
```

Requirements:

- Relation has persistent ID.
- Relation targets a concrete web-service distribution instance.
- `input_format` must exactly match one ORIGINAL format of distribution.
- Relation identifies one execution route.
- Conversion consists of one step only.
- Conversion chaining is unsupported.
- Client selects relation through `conversion_id`, not plugin ID.
- Downloadable-file distributions are never conversion inputs or outputs.

No conversion-relation description field is added. User context comes from:

- Input/output format values.
- Plugin name.
- Plugin description.

Native and converted outputs remain separate choices even when output format strings match.

---

## 7. Available-format discovery

Add computed `available_formats` to the web-service metadata shape returned by:

```http
GET /resources/{resourceType}/{id}/versions/{versionId}
```

Require the field on each web-service distribution's `web_service` object, including when only one native format is available. Compute it from the exact resource version returned by this endpoint. Do not add it to list, search or other detail endpoints.

Do not include `available_formats` for downloadable-file distributions. They retain their existing `format` metadata and never have conversions.

The OpenAPI schema defines `available_formats` as a list with a `oneOf` item schema discriminated by `kind`, whose enum is `[ORIGINAL, CONVERTED]`. Both shapes below are part of the target contract. Initial response implementation returns ORIGINAL entries only; add a TODO for CONVERTED entries when conversion support is implemented.

After conversion support exists, Gateway combines:

1. Native capability information from Metadata Lifecycle.
2. Applicable configured conversion relations.

Initially, Gateway returns only native capabilities obtained from Metadata Lifecycle; no plugin or conversion-relation lookup is performed.

### ORIGINAL entry

```json
{
  "kind": "ORIGINAL",
  "output_format": "geojson",
  "label": "GeoJSON",
  "is_default": true
}
```

- Enum label reused when available.
- Otherwise raw format value is display label.
- Fixed/default-only and controller-free choices use raw value.
- Exactly one ORIGINAL option, the native default, has `is_default=true`; all other ORIGINAL options have `is_default=false`.

### CONVERTED entry

```json
{
  "kind": "CONVERTED",
  "conversion_id": "conversion-uuid",
  "input_format": "xml_dc",
  "output_format": "geojson",
  "plugin": {
    "id": "plugin-uuid",
    "name": "Station converter",
    "description": "Converts station metadata to geospatial formats."
  }
}
```

Converted entries never become implicit default.

No separate available-format database IDs or opaque selection keys are needed:

- Native selection uses `output_format`.
- Converted selection uses `conversion_id`.

OpenAPI shape requirements:

- `available_formats` is a required, non-null array on the web-service object.
- ORIGINAL: require `kind`, `output_format`, `label` and `is_default`; `kind` is `ORIGINAL`; formats/label are strings and `is_default` is boolean.
- CONVERTED: require `kind`, `conversion_id`, `input_format`, `output_format` and `plugin`; `kind` is `CONVERTED`; IDs are UUID strings and format values are strings.
- `plugin` requires UUID string `id` and string `name`/`description`.
- Disallow additional properties in both entries and the plugin object.

### Future conversion availability rules (TODO)

- Compute options for selected metadata instance.
- Omit conversions known to be disabled.
- Omit conversions lacking installed/enabled plugin.
- Temporary runtime outages do not alter capability list.
- Revalidate selection during access; previously returned details are not authorization or availability guarantees.
- Discovery and execution use consistent capability rules.
- Do not build plugin/relation infrastructure or conversion availability checks in the initial implementation.

---

## 8. Component ownership

| Component          | Responsibility                                                                                                                            |
| ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Gateway            | Public API, orchestration, combined format discovery, error responses and final streaming.                                                |
| Metadata Lifecycle | Distribution/version resolution, visibility enforcement, value validation, defaults, template rendering and outbound request preparation. |
| Data Retriever     | Execute prepared HTTP request and stream response.                                                                                        |
| Future Converter   | Execute selected configured transformation and provide converted response metadata/content.                                               |

No separate service is introduced.

### Native flow

```text
Client
  → Gateway
    → Metadata Lifecycle: resolve, authorize, prepare
    → Data Retriever: execute, stream
  → Client
```

### Future converted flow

```text
Client
  → Gateway
    → Resolve selected conversion
    → Metadata Lifecycle: prepare selected native input
    → Data Retriever: execute, stream response to Gateway
    → Gateway: relay stream to Converter
    → Converter: buffer complete input, convert it, stream converted output to Gateway
    → Gateway: stream converted output to client
  → Client
```

Gateway relays data incrementally and does not buffer the complete response. Converter buffers the complete input and starts conversion only after retrieval completes.

Validate inputs and resolve executable selection before outbound retrieval.

---

## 9. Downloadable-file endpoint

### Endpoint

```http
GET /distributions/{distributionId}/download
```

This endpoint supports downloadable-file distributions only. It has no request body or query parameters.

### Resolution and execution

Resolve the specific downloadable-file distribution and its owning dataset version by distribution ID. Published versions permit anonymous access; every other state requires authentication and existing dataset access permissions. Unknown or deleted distribution IDs return 404; never fall back to another version or remap stale IDs.

Metadata Lifecycle prepares:

- GET request.
- Stored file URL.
- No caller parameters.

Requirements:

- File format remains available from distribution metadata; it is not an access selector.
- Same authorization, destination, redirect, timeout, size and response policies apply.
- Downloadable files are never converted.

---

## 10. Upstream credentials and request safety

### Credential support

All upstream inputs use the same generic parameter mechanism; there is no dedicated credential field, sensitivity flag or custom per-parameter handling. Do not log parameter values or add them to Gateway-generated Problem details. Upstream failure bodies are returned verbatim and may contain values echoed by the upstream service.

Platform authentication is separate and never copied upstream automatically.

Follow the HTTP client's standard redirect behavior for sensitive headers; no custom credential-forwarding framework is required.

Backend owns HTTP framing and must not allow caller input to bypass destination or transport safeguards.

### Destination policy

Enforce destination restrictions with Kubernetes network egress controls for Data Retriever pods, not with application-level IP lists, DNS checks or direct-IP dialing.

- Egress policy must block cluster and deployment-internal networks while allowing required public Internet access.
- Policy must account for cluster-specific internal networks and be enforced by the cluster networking layer.
- Do not use environment-configured proxies.

### Redirects

- Follow redirects using standard Go `http.Client` behavior.
- Maximum 10 redirects (go http client default).
- Kubernetes egress policy applies to every connection, including redirect hops.
- Use standard HTTP-client method semantics:
  - 301/302/303 may become GET and drop body.
  - 307/308 preserve method/body when replayable.
- Do not add automatic retries beyond redirect handling.

---

## 11. Streaming, timeouts and size limits

### Native streaming behavior

- Data Retriever sends bounded chunks.
- Gateway uses ogen's native `io.Reader` response streaming, reading and writing incrementally without explicit per-chunk flushing.
- Gateway does not buffer successful response bodies; sequential I/O uses existing transport flow control for backpressure.
- No custom queues, worker pools or idle-progress tracking are required.
- Client disconnect cancels retrieval and any active conversion.

### Total execution timeout

Gateway starts one configurable deadline when the access handler begins; default **60 seconds**. Propagate the same deadline through metadata preparation, retrieval and response streaming. Future conversion stages share it; never reset the timeout between stages.

Separate connection/header and streaming-idle timeouts are not required initially.

### Response-size limit

Configurable `MAX_RESPONSE_BYTES`, default **1 GiB**.

Apply independently to:

- Retriever output.
- Future converter output.

Count bytes incrementally as they pass through each stage. No separate wire-byte limit.

- Reject known oversized content before committing success headers when size information proves it exceeds limit.
- Otherwise enforce streaming counter.
- Stop pipeline when limit exceeded.
- Gateway does not buffer the response to determine size.
- Converter's complete-input buffer is subject to the same input size cap; its storage strategy and memory ceiling are deferred.

### Failure after streaming starts

Before response headers are committed, return Gateway Problem.

After success headers/body start:

- Abort stream.
- Cancel remaining pipeline.
- Record failure in telemetry.

---

## 12. Response policy

### Successful native responses

Stream upstream success status and body. Gateway owns HTTP framing.

No access-specific response-header forwarding is required initially. Do not derive headers from format selectors or infer download filenames.

Retrieval records remain internal; no public `X-Retrieval-Id`.

### Future converted responses

- Gateway streams retrieved input to the converter; the converter buffers the complete input and converts it after retrieval completes.
- Converted output is streamed back through Gateway to the client.
- Converter-specific response headers and metadata are deferred.
- Never pass upstream failure body through converter.
- Preserve upstream 204/205 without conversion because no content exists.

Detailed converter implementation/protocol remains deferred.

---

## 13. Failure contract

### General requirements

Use standard Gateway Problem response:

- Machine-readable `type`.
- Standard `title`.
- Client-safe `detail`.
- `instance`, containing the request trace ID.

Use a dedicated machine-readable Problem `type` for upstream HTTP failures. Other server-side failures use the standard internal-error Problem type; do not add a new type for every failure cause.

### Upstream HTTP failures

Upstream 4xx/5xx responses:

- Return HTTP 503 and preserve original upstream status in `upstream_response`.
- Return Gateway Problem, not raw upstream body.
- Use dedicated upstream-HTTP-failure type.
- Include original upstream status and body in `upstream_response`.
- Buffer the upstream failure body before returning the Problem; include it in full, subject to the response-size limit.

```json
{
  "type": "upstream_http_failure",
  "title": "Service Unavailable",
  "status": 503,
  "detail": "Upstream service returned an HTTP failure.",
  "instance": "<trace-id>",
  "upstream_response": {
    "status": 404,
    "body": "Requested station was not found."
  }
}
```

`upstream_response` appears only when an upstream HTTP failure response exists.

Assume upstream error bodies are text. No binary detection or special binary response is required. Do not parse upstream content as Gateway Problem or add request values/internal transport diagnostics to the Problem.

Do not log parameter values. Use safe generic details for server-side Problems.

### Status mapping

| Outcome                                                                                        |                                               HTTP status |
| ---------------------------------------------------------------------------------------------- | --------------------------------------------------------: |
| Invalid input, invalid selector or wrong distribution type                                     |                                                       400 |
| Authentication, authorization and not-found outcomes                                           | Reuse existing Metadata Lifecycle mappings (401/403/404). |
| Upstream HTTP failure                                                                          |                                                       503 |
| Total execution timeout                                                                        |                                                       504 |
| Other server-side failures, including retrieval, destination-policy and response-size failures |                                                       500 |

Existing visibility conventions take precedence over revealing inaccessible entities.

Other server-side failures use a safe generic detail and trace ID. Do not introduce exhaustive cause-specific HTTP mappings initially.

---

## 14. API scope and deferred request export

Public execution endpoints:

- `POST /distributions/{distributionId}/access` for web-service distributions.
- `GET /distributions/{distributionId}/download` for downloadable-file distributions.

Generic HTTP execution remains internal to Data Retriever. Do not expose a public generic `POST /access` endpoint. Retrieval records and identifiers remain internal.

Initial public access executes requests only. It has no mode selector, DRY response or request-export endpoint.

After real access is implemented, design and implement a way for clients to execute the equivalent upstream request themselves. Endpoint, representation (curl or otherwise), ownership and credential handling remain undecided. This capability is not necessarily part of Data Retriever.

Keep the OpenAPI specification, generated transport types, implementation and documentation aligned.

---

## 15. Explicitly deferred work

| Deferred item                         | Current boundary                                                                                                         |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| Converter implementation              | Native access ships first; conversion contracts prepared for future integration.                                         |
| Converter buffering implementation    | Full-input buffering is required; storage strategy and memory ceiling are decided with converter design.                 |
| Detailed converter protocol           | Success metadata/status rules, plugin execution and operational limits finalized with converter.                         |
| Stable cross-version identities       | Existing metadata-instance IDs retained; stale IDs fail.                                                                 |
| Conversion-relation copying/remapping | No automatic cross-version carry-forward defined.                                                                        |
| Successful response-header forwarding | No access-specific response-header forwarding initially; `Content-Type` and `Content-Disposition` are deferred.          |
| Client-executable request export      | Required later; design after real access. API shape, representation, ownership and credential handling remain undecided. |
| Conversion chaining                   | Unsupported; one native input → one converter only.                                                                      |

No conversion-relation description field, generated available-format ID, public plugin selector or MIME mapping is required by this design.
