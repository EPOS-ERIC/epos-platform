docker_compose('docker/compose.dev.yaml')

docker_build(
    'epos-gateway-dev:local',
    '.',
    dockerfile='services/gateway/Dockerfile.dev',
    only=[
        'services/gateway',
        'packages/go/logging',
        'docker/gateway-config.dev.yaml',
    ],
    live_update=[
        sync('services/gateway', '/workspace/services/gateway'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
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
    ],
    live_update=[
        sync('services/metadata-lifecycle/src', '/workspace/services/metadata-lifecycle/src'),
        sync('services/metadata-lifecycle/gradle', '/workspace/services/metadata-lifecycle/gradle'),
        sync('services/metadata-lifecycle/gradlew', '/workspace/services/metadata-lifecycle/gradlew'),
        sync('services/metadata-lifecycle/build.gradle.kts', '/workspace/services/metadata-lifecycle/build.gradle.kts'),
        sync('services/metadata-lifecycle/settings.gradle.kts', '/workspace/services/metadata-lifecycle/settings.gradle.kts'),
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
        'packages/go/migrations',
        'packages/go/postgres',
    ],
    live_update=[
        sync('services/user-group', '/workspace/services/user-group'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/go/migrations', '/workspace/packages/go/migrations'),
        sync('packages/go/postgres', '/workspace/packages/go/postgres'),
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
        'packages/go/migrations',
        'packages/go/postgres',
    ],
    live_update=[
        sync('services/data-retriever', '/workspace/services/data-retriever'),
        sync('packages/go/logging', '/workspace/packages/go/logging'),
        sync('packages/go/migrations', '/workspace/packages/go/migrations'),
        sync('packages/go/postgres', '/workspace/packages/go/postgres'),
        restart_container(),
    ],
)

dc_resource('user-group-db', labels=['infra'])
dc_resource('data-retriever-db', labels=['infra'])
dc_resource('neo4j', labels=['infra'])
dc_resource('metadata-lifecycle-service', labels=['app'], resource_deps=['neo4j'])
dc_resource('user-group-service', labels=['app'], resource_deps=['user-group-db'])
dc_resource('data-retriever-service', labels=['app'], resource_deps=['data-retriever-db'])
dc_resource('gateway-service', labels=['app'], resource_deps=['metadata-lifecycle-service'])
