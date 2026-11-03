docker_compose('docker/compose.dev.yaml')
docker_prune_settings(disable=False, max_age_mins=1, num_builds=1)

docker_build('epos-otel-lgtm-dev:local', 'docker', dockerfile='docker/Dockerfile.lgtm.dev', only=['Dockerfile.lgtm.dev'])

docker_build(
    'epos-gateway-dev:local',
    '.',
    dockerfile='services/gateway/Dockerfile.dev',
    only=[
        'services/gateway',
        'packages/go/apperror',
        'packages/go/logging',
        'packages/go/observability',
		'packages/protobuf',
        'packages/go/middleware',
        'packages/go/apidocs',
        'packages/scripts/epos-banner.sh',
        'docker/gateway-config.dev.yaml',
    ],
	ignore=[
        'services/gateway/.hypothesis',
    ],
    live_update=[
        sync('services/gateway', '/workspace/services/gateway'),
        sync('packages/go/apperror', '/workspace/packages/go/apperror'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/go/observability', '/workspace/packages/go/observability'),
        sync('packages/go/middleware', '/workspace/packages/go/middleware'),
        sync('packages/protobuf', '/workspace/packages/protobuf'),
        sync('packages/go/apidocs', '/workspace/packages/go/apidocs'),
        sync('packages/scripts/epos-banner.sh', '/usr/local/bin/epos-banner.sh'),
        sync('docker/gateway-config.dev.yaml', '/workspace/docker/gateway-config.dev.yaml'),
        restart_container(),
    ],
)

docker_build(
    'epos-metadata-lifecycle-dev:local',
    '.',
    dockerfile='services/metadata-lifecycle/Dockerfile.dev',
    only=[
        'services/metadata-lifecycle',
        'packages/go/apperror',
        'packages/go/grpcserver',
        'packages/go/logging',
        'packages/go/observability',
        'packages/go/postgres',
        'packages/protobuf',
        'packages/scripts/epos-banner.sh',
    ],
    ignore=[
        'services/metadata-lifecycle/db/sqlc',
    ],
    live_update=[
        sync('services/metadata-lifecycle', '/workspace/services/metadata-lifecycle'),
        sync('packages/go/apperror', '/workspace/packages/go/apperror'),
        sync('packages/go/grpcserver', '/workspace/packages/go/grpcserver'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/protobuf', '/workspace/packages/protobuf'),
        sync('packages/go/observability', '/workspace/packages/go/observability'),
        sync('packages/go/postgres', '/workspace/packages/go/postgres'),
        sync('packages/scripts/epos-banner.sh', '/usr/local/bin/epos-banner.sh'),
        restart_container(),
    ],
)

docker_build(
    'epos-sharing-dev:local',
    '.',
    dockerfile='services/sharing/Dockerfile.dev',
    only=[
        'services/sharing',
        'packages/go/apperror',
        'packages/go/grpcserver',
        'packages/go/logging',
        'packages/go/observability',
        'packages/go/postgres',
        'packages/protobuf',
        'packages/scripts/epos-banner.sh',
    ],
    live_update=[
        sync('services/sharing', '/workspace/services/sharing'),
        sync('packages/go/apperror', '/workspace/packages/go/apperror'),
        sync('packages/go/grpcserver', '/workspace/packages/go/grpcserver'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/go/observability', '/workspace/packages/go/observability'),
        sync('packages/go/postgres', '/workspace/packages/go/postgres'),
        sync('packages/protobuf', '/workspace/packages/protobuf'),
        sync('packages/scripts/epos-banner.sh', '/usr/local/bin/epos-banner.sh'),
        restart_container(),
    ],
)

docker_build(
    'epos-converter-dev:local',
    '.',
    dockerfile='services/converter/Dockerfile.dev',
    only=[
        'services/converter',
        'packages/go/apperror',
        'packages/go/grpcserver',
        'packages/go/logging',
        'packages/go/observability',
        'packages/go/postgres',
        'packages/protobuf',
        'packages/scripts/epos-banner.sh',
    ],
    live_update=[
        sync('services/converter', '/workspace/services/converter'),
        sync('packages/go/apperror', '/workspace/packages/go/apperror'),
        sync('packages/go/grpcserver', '/workspace/packages/go/grpcserver'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/go/observability', '/workspace/packages/go/observability'),
        sync('packages/go/postgres', '/workspace/packages/go/postgres'),
        sync('packages/protobuf', '/workspace/packages/protobuf'),
        sync('packages/scripts/epos-banner.sh', '/usr/local/bin/epos-banner.sh'),
        restart_container(),
    ],
)

docker_build(
    'epos-data-retriever-dev:local',
    '.',
    dockerfile='services/data-retriever/Dockerfile.dev',
    only=[
        'services/data-retriever',
        'packages/go/apperror',
        'packages/go/grpcserver',
        'packages/go/logging',
        'packages/go/observability',
        'packages/go/postgres',
        'packages/protobuf',
        'packages/scripts/epos-banner.sh',
    ],
    live_update=[
        sync('services/data-retriever', '/workspace/services/data-retriever'),
        sync('packages/go/apperror', '/workspace/packages/go/apperror'),
        sync('packages/go/grpcserver', '/workspace/packages/go/grpcserver'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/go/observability', '/workspace/packages/go/observability'),
        sync('packages/go/postgres', '/workspace/packages/go/postgres'),
        sync('packages/protobuf', '/workspace/packages/protobuf'),
        sync('packages/scripts/epos-banner.sh', '/usr/local/bin/epos-banner.sh'),
        restart_container(),
    ],
)

docker_build(
    'epos-oss-aai-dev:local',
    '.',
    dockerfile='services/oss-aai-service/Dockerfile.dev',
    only=[
        'services/oss-aai-service',
        'packages/go/logging',
        'packages/go/observability',
        'packages/scripts/epos-banner.sh',
    ],
    ignore=[
        'services/oss-aai-service/data',
    ],
    live_update=[
        sync('services/oss-aai-service', '/workspace/services/oss-aai-service'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/go/observability', '/workspace/packages/go/observability'),
        sync('packages/scripts/epos-banner.sh', '/usr/local/bin/epos-banner.sh'),
        restart_container(),
    ],
)

docker_build(
    'epos-platform-legacy-dev:local',
    '.',
    dockerfile='apps/platform-legacy/Dockerfile.dev',
    only=[
        'apps/platform-legacy',
        'packages/scripts/epos-banner.sh',
    ],
    ignore=[
        'apps/platform-legacy/.angular',
        'apps/platform-legacy/cypress',
        'apps/platform-legacy/dist',
        'apps/platform-legacy/node_modules',
    ],
    live_update=[
        sync('apps/platform-legacy/src', '/workspace/apps/platform-legacy/src'),
    ],
)

docker_build(
    'epos-backoffice-dev:local',
    '.',
    dockerfile='apps/backoffice/Dockerfile.dev',
    only=[
        'apps/backoffice',
        'packages/scripts/epos-banner.sh',
    ],
    ignore=[
        'apps/backoffice/.angular',
        'apps/backoffice/dist',
        'apps/backoffice/node_modules',
    ],
    live_update=[
        sync('apps/backoffice/src', '/workspace/apps/backoffice/src'),
        sync('apps/backoffice/public', '/workspace/apps/backoffice/public'),
        sync('apps/backoffice/generated', '/workspace/apps/backoffice/generated'),
    ],
)

local_resource(
    'backoffice-codegen',
    cmd='''docker run --rm --user "$(id -u):$(id -g)" -e HOME=/tmp \
        -v "$(pwd):/repo:ro" \
        -v "$(pwd)/apps/backoffice/generated:/output" \
        node:26 sh -c '
            mkdir -p /tmp/workspace/apps/backoffice /tmp/workspace/services/gateway/docs &&
            cp /repo/apps/backoffice/Makefile /repo/apps/backoffice/package.json /repo/apps/backoffice/pnpm-lock.yaml /repo/apps/backoffice/pnpm-workspace.yaml /repo/apps/backoffice/openapi-codegen.config.ts /tmp/workspace/apps/backoffice/ &&
            cp /repo/services/gateway/docs/openapi.yaml /tmp/workspace/services/gateway/docs/ &&
            cd /tmp/workspace/apps/backoffice && ln -s /output generated &&
            make PNPM="npx --yes pnpm@11.24.0" install generate
        ' ''',
    deps=[
        'services/gateway/docs/openapi.yaml',
        'apps/backoffice/openapi-codegen.config.ts',
        'apps/backoffice/Makefile',
        'apps/backoffice/package.json',
        'apps/backoffice/pnpm-lock.yaml',
        'apps/backoffice/pnpm-workspace.yaml',
    ],
    labels=['codegen'],
)

local_resource(
    'protobuf-codegen',
    cmd='''docker run --rm --user "$(id -u):$(id -g)" -e HOME=/tmp \
        -v "$(pwd)/packages/protobuf:/workspace:ro" \
        -v "$(pwd)/packages/protobuf/gen:/workspace/gen" \
        -w /workspace --entrypoint sh bufbuild/buf:1.75.0 \
        -c 'buf generate' ''',
    deps=[
        'packages/protobuf/configuration',
        'packages/protobuf/converter',
        'packages/protobuf/data_retriever',
        'packages/protobuf/metadata_lifecycle',
        'packages/protobuf/buf.yaml',
        'packages/protobuf/buf.gen.yaml',
        'packages/protobuf/buf.lock',
    ],
    labels=['codegen'],
)

local_resource(
    'gateway-codegen',
    cmd='''mkdir -p "$HOME/.cache/epos-tilt-codegen" && docker run --rm --user "$(id -u):$(id -g)" \
        -e HOME=/tmp -e GOMODCACHE=/cache/mod -e GOCACHE=/cache/build \
        -v "$HOME/.cache/epos-tilt-codegen:/cache" \
        -v "$(pwd):/workspace:ro" \
        -v "$(pwd)/services/gateway/gen:/workspace/services/gateway/gen" \
        -w /workspace/services/gateway golang:1.27-alpine go generate .''',
    deps=[
        'services/gateway/docs/openapi.yaml',
        'services/gateway/.ogen.yml',
        'services/gateway/generate.go',
        'services/gateway/go.mod',
        'services/gateway/go.sum',
    ],
    labels=['codegen'],
)

for service in ['metadata-lifecycle', 'sharing', 'data-retriever', 'converter']:
    local_resource(
        service + '-sqlc',
        cmd='''mkdir -p "$HOME/.cache/epos-tilt-codegen" && docker run --rm --user "$(id -u):$(id -g)" \
            -e HOME=/tmp -e GOMODCACHE=/cache/mod -e GOCACHE=/cache/build \
            -v "$HOME/.cache/epos-tilt-codegen:/cache" \
            -v "$(pwd):/workspace:ro" \
            -v "$(pwd)/services/{service}/db/sqlc:/workspace/services/{service}/db/sqlc" \
            -w /workspace/services/{service} golang:1.27-alpine go generate ./db'''.format(service=service),
        deps=[
            'services/' + service + '/db/queries',
            'services/' + service + '/db/migrations',
            'services/' + service + '/sqlc.yaml',
            'services/' + service + '/go.mod',
            'services/' + service + '/go.sum',
        ],
        labels=['codegen'],
    )

dc_resource('data-retriever-db', labels=['infra'])
dc_resource('metadata-lifecycle-db', labels=['infra'])
dc_resource('sharing-db', labels=['infra'])
dc_resource('converter-db', labels=['infra'])
dc_resource('data-retriever-service', labels=['app'], resource_deps=['data-retriever-db', 'protobuf-codegen', 'data-retriever-sqlc'])
dc_resource('oss-aai-service', labels=['app'])
dc_resource('gateway-service', labels=['app'], resource_deps=['protobuf-codegen', 'gateway-codegen'])
dc_resource('metadata-lifecycle-service', labels=['app'], resource_deps=['metadata-lifecycle-db', 'protobuf-codegen', 'metadata-lifecycle-sqlc'])
dc_resource('sharing-service', labels=['app'], resource_deps=['sharing-db', 'protobuf-codegen', 'sharing-sqlc'])
dc_resource('converter-service', labels=['app'], resource_deps=['converter-db', 'protobuf-codegen', 'converter-sqlc'])
dc_resource('otel-lgtm', labels=['observability'])
dc_resource('platform-legacy', labels=['app'])
dc_resource('backoffice', labels=['app'], resource_deps=['backoffice-codegen'])
