# EPOS Platform Web App

Angular frontend for the EPOS Platform. It uses generated TypeScript types from the gateway OpenAPI contract and proxies API requests under `/v1` to the gateway during development.

## Requirements

- Node.js compatible with the version used by CI and the development image
- pnpm 11.4.0 (declared in `package.json`)

Install dependencies from this directory with `pnpm install`.

## Development

Start the development server and regenerate API types first:

```bash
pnpm run dev
```

Open `http://localhost:4200`. By default, `/v1` requests are proxied to `http://localhost:32000`. Override the target when needed:

```bash
PLATFORM_API_PROXY_TARGET=http://localhost:5000 pnpm run dev
```

The root `make up` command runs the app in the Docker Compose/Tilt development stack with live updates.

## API types

Regenerate `src/app/api/generated/schema.ts` after changing `services/gateway/docs/openapi.yaml`:

```bash
make codegen
```

Do not edit files under `src/app/api/generated/` manually.

## Build and checks

```bash
make build
make test
make fmt
make lint
```

- `make build` creates the production bundle under `dist/`.
- `make test` runs the Angular unit tests once with Vitest.
- `make fmt` applies Prettier.
- `make lint` formats and then checks the resulting files with Prettier.

Build the production container with `make docker`. Set `IMAGE` to override the default image name and tag.
