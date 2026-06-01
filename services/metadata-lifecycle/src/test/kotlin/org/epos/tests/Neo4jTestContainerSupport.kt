package org.epos.tests

import org.junit.jupiter.api.BeforeAll
import org.junit.jupiter.api.TestInstance
import org.testcontainers.containers.GenericContainer
import org.testcontainers.containers.wait.strategy.Wait
import org.testcontainers.utility.DockerImageName
import java.io.File
import java.time.Duration

private object SharedNeo4jTestRuntime {
    private var started = false
    private lateinit var configDir: File

    private val neo4jContainer =
        GenericContainer(DockerImageName.parse("neo4j:5.26-community"))
            .withExposedPorts(7687)
            .withEnv("NEO4J_AUTH", "none")
            .waitingFor(Wait.forLogMessage(".*Bolt enabled on.*", 1))
            .withStartupTimeout(Duration.ofMinutes(2))

    @Synchronized
    fun startIfNeeded() {
        if (started) return
        neo4jContainer.start()
        configDir = kotlin.io.path.createTempDirectory("metadata-lifecycle-test-config").toFile()
        configDir.resolve("application-test.yaml").writeText(
            """
            neo4j:
              uri: "bolt://${neo4jContainer.host}:${neo4jContainer.getMappedPort(7687)}"
              username: ""
              password: ""
              maxConnectionPoolSize: 10
              connectionTimeoutMs: 5000

            indexUrl: "file:///test/fixtures/index.txt"
            baseUri: "https://test.epos-eu.org"

            logging:
              level: "WARN"
              format: "json"

            features:
              enableSwagger: false
              enableCors: false
              enableMetrics: false
            """.trimIndent(),
        )
        System.setProperty("epos.profile", "test")
        System.setProperty("epos.config.dir", configDir.absolutePath)
        started = true
    }
}

@TestInstance(TestInstance.Lifecycle.PER_CLASS)
abstract class Neo4jTestContainerSupport {

    @BeforeAll
    fun startNeo4jContainer() {
        SharedNeo4jTestRuntime.startIfNeeded()
    }
}
