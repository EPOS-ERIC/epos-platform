docker_compose('docker/compose.dev.yaml')

docker_build(
    'epos-platform-dev:local',
    '.',
    dockerfile='apps/platform/Dockerfile.dev',
    only=[
        'apps/platform',
        'services/gateway/docs/openapi.yaml',
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
        'packages/go/middleware',
        'packages/go/apidocs',
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
        sync('packages/go/apidocs', '/workspace/packages/go/apidocs'),
        sync('docker/gateway-config.dev.yaml', '/workspace/docker/gateway-config.dev.yaml'),
        restart_container(),
    ],
)

docker_build(
    'epos-metadata-lifecycle-dev:local',
    '.',
    dockerfile='services/metadata-lifecycle/Dockerfile.dev',
    only=[
        'services/metadata-lifecycle/src',
        'services/metadata-lifecycle/gradle',
        'services/metadata-lifecycle/gradlew',
        'services/metadata-lifecycle/build.gradle.kts',
        'services/metadata-lifecycle/settings.gradle.kts',
        'packages/assets/otel/opentelemetry-javaagent.jar',
    ],
    live_update=[
        sync('services/metadata-lifecycle/src', '/workspace/services/metadata-lifecycle/src'),
        sync('services/metadata-lifecycle/gradle', '/workspace/services/metadata-lifecycle/gradle'),
        sync('services/metadata-lifecycle/gradlew', '/workspace/services/metadata-lifecycle/gradlew'),
        sync('services/metadata-lifecycle/build.gradle.kts', '/workspace/services/metadata-lifecycle/build.gradle.kts'),
        sync('services/metadata-lifecycle/settings.gradle.kts', '/workspace/services/metadata-lifecycle/settings.gradle.kts'),
        sync('packages/assets/otel/opentelemetry-javaagent.jar', '/otel/opentelemetry-javaagent.jar'),
        restart_container(),
    ],
)

docker_build(
    'epos-user-group-dev:local',
    '.',
    dockerfile='services/user-group/Dockerfile.dev',
    only=[
        'services/user-group',
        'packages/go/logging',
        'packages/go/observability',
        'packages/go/middleware',
        'packages/go/apidocs',
    ],
    live_update=[
        sync('services/user-group', '/workspace/services/user-group'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/go/observability', '/workspace/packages/go/observability'),
        sync('packages/go/middleware', '/workspace/packages/go/middleware'),
        sync('packages/go/apidocs', '/workspace/packages/go/apidocs'),
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
    ],
    live_update=[
        sync('services/data-retriever', '/workspace/services/data-retriever'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/go/observability', '/workspace/packages/go/observability'),
        sync('packages/go/migrations', '/workspace/packages/go/migrations'),
        sync('packages/go/postgres', '/workspace/packages/go/postgres'),
        restart_container(),
    ],
)

docker_build(
    'epos-oss-aai-dev:local',
    '.',
    dockerfile='services/oss-aai-service/Dockerfile.dev',
    only=[
        'services/oss-aai-service',
    ],
    live_update=[
        sync('services/oss-aai-service', '/workspace/services/oss-aai-service'),
        restart_container(),
    ],
)

dc_resource('user-group-db', labels=['infra'])
dc_resource('data-retriever-db', labels=['infra'])
dc_resource('neo4j', labels=['infra'])
dc_resource('metadata-lifecycle-service', labels=['app'], resource_deps=['neo4j'])
dc_resource('user-group-service', labels=['app'], resource_deps=['user-group-db'])
dc_resource('data-retriever-service', labels=['app'], resource_deps=['data-retriever-db'])
dc_resource('oss-aai-service', labels=['app'])
dc_resource('gateway-service', labels=['app'], resource_deps=['metadata-lifecycle-service'])
dc_resource('platform-app', labels=['app'], resource_deps=['gateway-service'])
dc_resource('otel-lgtm', labels=['observability'])
