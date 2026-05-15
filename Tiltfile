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
        run('cd /workspace/services/gateway && go mod download', trigger=['services/gateway/go.mod', 'services/gateway/go.sum']),
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

dc_resource('graphdb', labels=['infra'])
dc_resource('graphdb-init', labels=['infra'], resource_deps=['graphdb'])
dc_resource('metadata-lifecycle-service', labels=['app'], resource_deps=['graphdb-init'])
dc_resource('gateway-service', labels=['app'], resource_deps=['metadata-lifecycle-service'])
