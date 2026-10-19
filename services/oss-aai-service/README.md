# OSS AAI Service

Small Go application that provides:

- User registration and login (HTML templates)
- Session-based web authentication
- OAuth2/OIDC-style token issuance, introspection and user info
- Local persistence with SQLite (`data/auth.db`)

## Quick start

```bash
make build
make run
```

Open: `http://localhost:8080`

The repository Tilt stack publishes the same service at `http://localhost:35000`.

## Run with Docker

From this directory:

```bash
make docker
docker run --rm -p 8080:8080 epos-oss-aai-service:latest
```

If you want persistent local data when using Docker:

```bash
docker run --rm -p 8080:8080 -v "$(pwd)/data:/app/data" epos-oss-aai-service:latest
```

## Default seeded accounts

- Initial user (created once):
  - Name: `Admin`
  - Surname: `Admin`
  - Email: `admin@admin.org`
  - Password: `adminadmin`
- OAuth client (confidential/password grant):
  - `client_id`: `local-dev-client`
  - `client_secret`: `dev-secret`
- OIDC client (public SPA / `epos-backoffice-gui`):
  - `client_id`: `eposICS`
  - Allowed redirect URIs:
    - `http://localhost:34000/last-page-redirect`
    - `http://localhost:34000/silent-token-refresh.html`
    - `http://localhost:4200/testpath/last-page-redirect`
    - `http://localhost:4200/testpath/silent-token-refresh.html`
    - `http://localhost:32000/last-page-redirect`

The initial admin user can be updated or deleted after first login and will not be recreated.

## Environment variables

- `INITIAL_ADMIN_NAME` - first name used for the seeded admin user (only on first initialization)
- `INITIAL_ADMIN_SURNAME` - surname used for the seeded admin user (only on first initialization)
- `INITIAL_ADMIN_EMAIL` - email/login identifier used for the seeded admin user (only on first initialization)
- `INITIAL_ADMIN_PASSWORD` - password used for the seeded admin user (only on first initialization)
- `APP_SECURE_COOKIES=true` - forces the `Secure` cookie flag (recommended behind HTTPS)
- `APP_CORS_ALLOW_ORIGIN` - when set, adds the `Access-Control-Allow-Origin` header with the provided value and enables basic CORS preflight handling
- `OSS_AAI_LISTEN_ADDR` - HTTP listen address, defaults to `:8080`
- `OIDC_ISSUER` - overrides issuer URL in OIDC discovery metadata

If none of the `INITIAL_ADMIN_*` variables are set before the first startup, the service seeds the default `Admin Admin / admin@admin.org / adminadmin` account.

## Main endpoints

- Health:
  - `GET /health`
  - `GET /healthz`
- Web UI:
  - `GET /register`
  - `GET /login`
  - `GET /` (requires login)
- OAuth/API:
  - `POST /oauth/token`
  - `GET /oauth/validate`
  - `GET /api/me`
  - `GET /oauth2/authorize`
  - `POST /oauth2/token`
  - `POST /oauth2/introspect`
  - `POST /oauth2/revoke`
  - `GET /oauth2/userinfo`
  - `GET /oauth2/jwk`
  - `GET /oauth2/.well-known/openid-configuration`
  - `GET /.well-known/openid-configuration`
  - `GET /oauth2-as/oauth2-authz` (legacy auth alias)

## Health check example

```bash
curl http://localhost:8080/healthz
```

## OIDC discovery example

```bash
curl http://localhost:8080/oauth2/.well-known/openid-configuration
```

The validation/introspection response includes:

- `active`
- `eduPersonUniqueId`
- `firstname`
- `lastName`
- `email`
- `exp`

## Browser OIDC flow example

```bash
curl "http://localhost:8080/oauth2/authorize?response_type=id_token%20token&client_id=eposICS&redirect_uri=http%3A%2F%2Flocalhost%3A34000%2Flast-page-redirect&scope=openid%20profile%20single-logout&state=demo-state&nonce=demo-nonce"
```

If the browser already has a valid web session, the service redirects back to the GUI callback URI with an `access_token` and signed `id_token` in the fragment.

## Token example

```bash
curl -X POST http://localhost:8080/oauth/token \
  -u local-dev-client:dev-secret \
  -d "grant_type=password" \
  -d "username=admin@admin.org" \
  -d "password=adminadmin"
```

## Make targets

- `make run` - run app
- `make build` - build binary to `bin/oss-aai-service`
- `make test` - run tests
- `make fmt` - format Go files
- `make tidy` - tidy Go modules
- `make clean` - remove `bin/`
