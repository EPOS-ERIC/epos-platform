docker_compose('docker/compose.dev.yaml')
docker_prune_settings(disable=False, max_age_mins=1, num_builds=1)

docker_build(
    'epos-platform-dev:local',
    '.',
    dockerfile='apps/platform/Dockerfile.dev',
    only=[
        'apps/platform',
        'services/gateway/docs/openapi.yaml',
        'packages/scripts/epos-banner.sh',
    ],
    ignore=[
        'apps/platform/.angular',
        'apps/platform/dist',
        'apps/platform/node_modules',
    ],
    live_update=[
        sync('apps/platform/src', '/workspace/apps/platform/src'),
        sync('apps/platform/public', '/workspace/apps/platform/public'),
        sync('apps/platform/angular.json', '/workspace/apps/platform/angular.json'),
        sync('apps/platform/package.json', '/workspace/apps/platform/package.json'),
        sync('services/gateway/docs/openapi.yaml', '/workspace/services/gateway/docs/openapi.yaml'),
        sync('packages/scripts/epos-banner.sh', '/usr/local/bin/epos-banner.sh'),
        restart_container(),
    ],
)

docker_build(
    'epos-gateway-dev:local',
    '.',
    dockerfile='services/gateway/Dockerfile.dev',
    only=[
        'services/gateway',
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
        'packages/go/logging',
        'packages/go/observability',
		'packages/protobuf',
        'packages/go/middleware',
        'packages/go/apidocs',
        'packages/scripts/epos-banner.sh',
    ],
    live_update=[
        sync('services/metadata-lifecycle', '/workspace/services/metadata-lifecycle'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/protobuf', '/workspace/packages/protobuf'),
        sync('packages/go/observability', '/workspace/packages/go/observability'),
        sync('packages/go/middleware', '/workspace/packages/go/middleware'),
        sync('packages/go/apidocs', '/workspace/packages/go/apidocs'),
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
        'packages/go/logging',
        'packages/go/observability',
        'packages/go/migrations',
        'packages/go/postgres',
        'packages/scripts/epos-banner.sh',
    ],
    live_update=[
        sync('services/data-retriever', '/workspace/services/data-retriever'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/go/observability', '/workspace/packages/go/observability'),
        sync('packages/go/migrations', '/workspace/packages/go/migrations'),
        sync('packages/go/postgres', '/workspace/packages/go/postgres'),
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
        'packages/scripts/epos-banner.sh',
    ],
    live_update=[
        sync('services/oss-aai-service', '/workspace/services/oss-aai-service'),
        sync('packages/scripts/epos-banner.sh', '/usr/local/bin/epos-banner.sh'),
        restart_container(),
    ],
)

dc_resource('data-retriever-db', labels=['infra'])
dc_resource('metadata-lifecycle-db', labels=['infra'])
dc_resource('data-retriever-service', labels=['app'], resource_deps=['data-retriever-db'])
dc_resource('oss-aai-service', labels=['app'])
dc_resource('gateway-service', labels=['app'])
dc_resource('metadata-lifecycle-service', labels=['app'], resource_deps=['metadata-lifecycle-db'])
dc_resource('platform-app', labels=['app'], resource_deps=['gateway-service'])
dc_resource('otel-lgtm', labels=['observability'])
