# EPOS Backoffice GUI

Angular frontend for browsing, creating, editing, and reviewing EPOS catalogue resources. The application also provides administration pages for groups, organizations, and categories.

This README is a living architecture overview. Keep it aligned with the application when responsibilities move between components, stores, services, or mappers.

## Technology

- Angular 21 with standalone components and signals
- RxJS for asynchronous workflows
- OAuth 2/OIDC authentication
- OpenAPI-generated TypeScript contracts
- Vitest through the Angular test builder
- Leaflet for spatial editors

## Project Structure

```text
epos-backoffice-gui/
├── generated/        # OpenAPI-generated API contracts
├── public/           # Static files copied as-is into the build
└── src/
    ├── app/          # Application bootstrap, routes, and guards
    ├── assets/       # Application-managed static assets
    ├── components/   # Routed pages, page sections, and reusable UI
    ├── mappers/      # Pure API DTO and editable-form transformations
    ├── models/       # UI models, capabilities, and shared types
    ├── services/     # API access and application-wide workflows
    ├── stores/       # Scoped state and UI orchestration
    ├── types/        # Type declarations for external libraries
    └── styles.css    # Global theme tokens and shared UI primitives
```

Tests are colocated with the code they cover and use the `*.spec.ts` suffix.

## Main Building Blocks

### Application Root

`src/app/app.ts` mounts the router and the global dialog, resource-creation dialog, and toast outlets. Authentication and group initialization are registered in `src/app/app.config.ts`.

### Main Container

`MainContainerComponent` is the routed application shell. It coordinates:

- the navigation sidebar and page header;
- route mode selection between results, resource detail, and administration;
- discovery filters and result loading;
- theme selection and top-level resource creation.

Resource-specific form and lifecycle logic must not be added back to this component.

### Resource Detail

`ResourceDetailPageComponent` composes the dataset and software detail editors. It reads state and delegates user operations to `ResourceDetailStore`.

`ResourceDetailStore` is provided by `MainContainerComponent`, so each routed shell receives an isolated store instance. It owns:

- the current resource version and editable form;
- loading and error state;
- lifecycle capabilities and sidebar actions;
- save, submit, publish, reject, archive, reopen, draft creation, and deletion workflows;
- reload and post-action navigation.

Draft details can open the version-history dialog. The store loads discovery entries for the current group and resource, limits comparison selection to two versions, and navigates to the section-specific comparison route.

Detail loads use `switchMap`. Starting another load or clearing the store cancels the previous request and prevents stale responses from replacing the current route state.

### Version Comparison

`ResourceVersionComparisonPageComponent` loads the two selected versions through its component-scoped `ResourceVersionComparisonStore`. The pure `resource-version-diff.mapper.ts` flattens both API responses into a field-by-field side-by-side diff and marks changed values without modifying the source payloads.

### Results And Administration

`ResultsTableComponent` renders discovery results and owns client-side pagination. Discovery filters and API loading currently remain in the main container.

Administration pages are independent components under `src/components/admin-*` and use `ApiService` for their domain operations.

## Data Flow

The main resource-detail flow is:

```text
Angular route
    -> MainContainerComponent
    -> ResourceDetailStore
    -> ApiService / ResourceActionsService
    -> ResourceDetailPageComponent and navigation sidebar
    -> Version History dialog
    -> ResourceVersionComparisonPageComponent
```

The main results flow is:

```text
Header filters
    -> MainContainerComponent
    -> ApiService discovery request
    -> ResultsTableComponent
```

Components should emit user intent or local model changes. API orchestration belongs in a scoped store or a service, not in presentation components.

## State Management

Use the narrowest suitable state scope:

- Component-local state for isolated visual interaction.
- Component-scoped stores for state shared by a page and its children.
- Root-provided services or stores only for application-wide state and infrastructure.

`ResourceDetailStore` is component-scoped. `GroupsStore`, authentication, dialogs, toast notifications, and API access are application-wide.

## Forms And Mapping

Resource editors currently use typed plain-object view models rather than Angular reactive forms.

Dataset form types are defined in `src/models/dataset-detail-form.ts`. Pure transformations and validation are kept in:

- `src/mappers/dataset-detail-form.mapper.ts`
- `src/mappers/software-detail-form.mapper.ts`

Mappers must remain independent of Angular dependency injection and component state. They receive all required form, revision, and preserved metadata values explicitly and are covered by characterization tests.

## Capabilities And Authorization

`src/models/resource-capabilities.ts` contains the pure lifecycle capability matrix used by the detail store and action presentation.

UI capabilities determine which controls are visible or enabled. They do not replace backend authorization. The backend remains the security boundary for every operation.

## API Integration

`ApiService` is the frontend boundary for HTTP operations. The development proxy maps `/api` to the local backend at `http://localhost:33000`.

Contracts under `generated/` are produced from the backend OpenAPI document:

```bash
make generate
```

The generator reads `services/gateway/docs/openapi.yaml` from the repository. Never edit generated files manually; update the API specification and regenerate them instead.

The published Nginx image serves the built app and proxies `/api/` to `API_HOST` (default `http://gateway:5000`), stripping the `/api/` prefix to match the Gateway `/v1` routes.

## UI Infrastructure

Shared theme tokens and primitives such as cards, buttons, inputs, badges, and dialogs are defined in `src/styles.css`. Component styles should build on these primitives and remain encapsulated when they are specific to one component.

Global overlays are mounted once by the application root and controlled through their corresponding services:

- `DialogService`
- `ResourceCreationDialogService`
- `ToastService`

## Development

The repository-level Tilt configuration runs the backoffice at `http://localhost:34000`, with API requests proxied to the local Gateway and source changes live-reloaded. Tilt runs code generation in Docker at startup and whenever the Gateway OpenAPI specification changes, writing contracts to `generated/` on the host.

For module-local frontend work:

```bash
make install
make dev
```

`make dev` regenerates the API contracts before starting the Angular development server.

Useful module commands:

```bash
pnpm start                         # Start Angular without running code generation
make build                         # Production build
make test                          # Run the test suite once
make generate                      # Regenerate OpenAPI contracts
```

## Architecture Guidelines

- Keep routed shell components focused on composition and page selection.
- Do not import models from parent components; place shared types under `src/models`.
- Prefer pure functions for deterministic mapping, normalization, validation, and policy decisions.
- Use Angular services or stores only when dependency injection, asynchronous orchestration, or shared state is required.
- Keep page-specific stores component-scoped unless state must intentionally survive navigation.
- Extend existing components and services when their responsibility already matches the change.
- Add characterization tests before moving behavior-heavy mapping or workflow code.
- Preserve API timestamps unless a view explicitly requires timezone conversion.
- Update this README when the application structure or responsibility boundaries change.
