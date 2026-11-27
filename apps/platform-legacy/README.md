# EPOS GUI

Essential commands and runtime config for local development and Docker deployment.

## Development

| Task                                 | Command           | Notes                                         |
| ------------------------------------ | ----------------- | --------------------------------------------- |
| Install dependencies                 | `make install`    | Run once after checkout or dependency updates |
| Start open source development server | `make dev-oss`    | Local URL: `http://localhost:4200/testpath`   |
| Start internal development server    | `make dev`        | Local URL: `http://localhost:4200/testpath`   |
| Open Cypress runner                  | `make cypress`    | Interactive Cypress launcher                  |

## Quality checks

| Task                            | Command                                                | Notes                              |
| ------------------------------- | ------------------------------------------------------ | ---------------------------------- |
| Run lint                        | `make lint`                                            | Runs Angular ESLint rules          |
| Run headless e2e tests          | `make test`                                            | Runs Cypress in Chrome             |
| Run one Cypress spec (headless) | `npm run test -- --spec "cypress/e2e/your_spec.cy.ts"` | Fast feedback for targeted changes |

## Build

| Task                                        | Command                               | Notes                                                   |
| ------------------------------------------- | ------------------------------------- | ------------------------------------------------------- |
| Create open source build                    | `make build-oss`                      | CI uses this build flavor                               |
| Create internal production build            | `make build`                          | Internal configuration                                  |
| CI-like build (includes metadata injection) | `./pre-build.sh && make build-oss`    | Modifies `src/environments/environmentBase.ts` in place |

Build output is written to `dist/`.

## Data portal concept tabs

The data portal creates its sidebar tabs in the order returned by `GET /api/v1/concepts` and starts a search for each concept ID. Data, Facilities, and Software use their specialized panels when the corresponding module is enabled. Other tabs display the resource names returned by `GET /api/v1/search?conceptId=<id>` and remain available when the result list is empty.

## Dataset web service formats

HTTP, DYNAMIC and WFS use the flattened `web_service.available_formats`. Each download row retains its exact native format or conversion ID and plugin; provider URL preparation supports native formats only.

WMS and WMTS use internal `SERVICE` descriptors to select map adapters, without advertising provider output formats or conversions. They access providers directly. WMS map and download image formats are checked against GetCapabilities; WMTS retains the existing TileJSON map path and does not advertise a whole-service download.

Provider query parameters and the separate map/tile and GetFeatureInfo parameter sets are retained. Operation inputs use parameter IDs internally to avoid collisions between operations. HTTP/DYNAMIC/WFS templates remain rendered by Gateway. Direct WMS/WMTS operation templates support `{{.name}}` field substitution; other Go template expressions produce an explicit error rather than an incorrectly rendered provider request.

## Docker configuration

The container image supports runtime configuration via environment variables.

| Variable        | Default                   | Description                                                                                 |
| --------------- | ------------------------- | ------------------------------------------------------------------------------------------- |
| `BASE_URL`      | `/`                       | Base path where the app is served. Must start and end with `/` (for example `/gui/`).      |
| `API_HOST`      | `http://gateway:5000/api` | Upstream API URL used by nginx for `/api` requests.                                         |
| `AUTH_ROOT_URL` | `http://localhost:35000`  | Browser-visible OAuth root URL injected into the OSS build at container startup. Internal builds keep `https://aaai.epos-eu.org`. |
| `SERVER_NAME`   | `_`                       | nginx `server_name` value.                                                                  |

Example:

```bash
docker run --rm -p 8080:80 \
  -e BASE_URL=/gui/ \
  -e API_HOST=https://api.example.org/ \
  -e AUTH_ROOT_URL=https://aaai.epos-eu.org \
  -e SERVER_NAME=_ \
  epos-gui:latest
```
