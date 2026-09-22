docker_compose('docker/compose.dev.yaml')
docker_prune_settings(disable=False, max_age_mins=1, num_builds=1)

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
        'packages/scripts/epos-banner.sh',
    ],
    ignore=[
        'services/metadata-lifecycle/db/sqlc',
    ],
    live_update=[
        sync('services/metadata-lifecycle', '/workspace/services/metadata-lifecycle'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/protobuf', '/workspace/packages/protobuf'),
        sync('packages/go/observability', '/workspace/packages/go/observability'),
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
        'packages/go/postgres',
        'packages/protobuf',
        'packages/scripts/epos-banner.sh',
    ],
    live_update=[
        sync('services/data-retriever', '/workspace/services/data-retriever'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/go/observability', '/workspace/packages/go/observability'),
        sync('packages/go/postgres', '/workspace/packages/go/postgres'),
        sync('packages/protobuf', '/workspace/packages/protobuf'),
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
        'packages/go/logging',
        'packages/go/observability',
        'packages/go/postgres',
        'packages/protobuf',
        'packages/scripts/epos-banner.sh',
    ],
    live_update=[
        sync('services/sharing', '/workspace/services/sharing'),
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
        'packages/scripts/epos-banner.sh',
    ],
    ignore=[
        'services/oss-aai-service/data',
    ],
    live_update=[
        sync('services/oss-aai-service', '/workspace/services/oss-aai-service'),
        sync('packages/scripts/epos-banner.sh', '/usr/local/bin/epos-banner.sh'),
        restart_container(),
    ],
)

dc_resource('data-retriever-db', labels=['infra'])
dc_resource('metadata-lifecycle-db', labels=['infra'])
dc_resource('sharing-db', labels=['infra'])
dc_resource('data-retriever-service', labels=['app'], resource_deps=['data-retriever-db'])
dc_resource('sharing-service', labels=['app'], resource_deps=['sharing-db'])
dc_resource('oss-aai-service', labels=['app'])
dc_resource('gateway-service', labels=['app'], resource_deps=['data-retriever-service', 'sharing-service'])
dc_resource('metadata-lifecycle-service', labels=['app'], resource_deps=['metadata-lifecycle-db'])
dc_resource('otel-lgtm', labels=['observability'])
