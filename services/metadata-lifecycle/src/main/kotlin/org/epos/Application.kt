package org.epos

import io.ktor.client.*
import io.ktor.server.application.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.routing.*
import kotlinx.coroutines.runBlocking
import net.logstash.logback.argument.StructuredArguments.entries
import org.epos.api.controllers.ResourceController
import org.epos.di.appModule
import org.epos.propertygraph.Config
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.services.ProvenanceService
import org.epos.propertygraph.services.population.InitialPopulationService
import org.epos.propertygraph.services.population.RelationReconciliationService
import org.epos.propertygraph.services.population.RevisionImportService
import org.epos.infrastructure.ktor.plugins.*
import org.epos.infrastructure.ktor.routes.*
import org.epos.infrastructure.ktor.routes.search.*
import org.epos.services.ShaclService
import org.koin.core.context.startKoin
import org.koin.core.context.stopKoin
import org.koin.ktor.ext.inject
import org.koin.ktor.plugin.Koin
import org.koin.logger.slf4jLogger
import org.slf4j.LoggerFactory

private val logger = LoggerFactory.getLogger("org.epos.Application")

fun main(args: Array<String>) = runBlocking {
    val mode = args.firstOrNull()?.lowercase() ?: "server"

    when (mode) {
        "server", "serve" -> {
            logger.info("Starting EPOS API Server...")
            startServer()
        }
        "populate", "init" -> {
            logger.info("Running Initial Population...")
            runPopulation()
        }
        "populate-and-serve", "init-and-serve", "all" -> {
            logger.info("Running Initial Population...")
            runPopulation()
            logger.info("Starting EPOS API Server...")
            startServer()
        }
        "--help", "-h", "help" -> {
            printHelp()
        }
        else -> {
            logger.warn("Unknown mode: {}", mode)
            printHelp()
        }
    }
}

private fun printHelp() {
    println(
        """
        |=================================================
        |   EPOS Neo4j Metadata Lifecycle Management System
        |=================================================
        |
        |Usage: java -jar app.jar [mode]
        |
        |Modes:
        |  server              Start the REST API server (default)
        |  populate            Run initial database population only
        |  populate-and-serve  Run population, then start server
        |  --help              Show this help message
        |
        |Environment Variables:
        |  EPOS_NEO4J_URI         Neo4j Bolt URI
        |  EPOS_NEO4J_USERNAME    Neo4j username
        |  EPOS_NEO4J_PASSWORD    Neo4j password
        |  EPOS_NEO4J_DATABASE    Neo4j database
        |  EPOS_INDEX_URL       URL to index.txt file
        |  EPOS_BASE_URI        Base URI for generated resources
        |  PORT                   Server port (default: 8080)
        |  HOST                   Server host (default: 0.0.0.0)
        |
        |Examples:
        |  java -jar app.jar server
        |  java -jar app.jar populate
        |  EPOS_NEO4J_URI=bolt://prod:7687 java -jar app.jar populate-and-serve
        |
    """.trimMargin()
    )
}

private suspend fun runPopulation() {
    startKoin {
        slf4jLogger()
        modules(appModule)
    }

    val httpClient = org.koin.core.context.GlobalContext.get().get<HttpClient>()
    val client = org.koin.core.context.GlobalContext.get().get<PropertyGraphClient>()
    val provenanceService = org.koin.core.context.GlobalContext.get().get<ProvenanceService>()
    val shaclService = org.koin.core.context.GlobalContext.get().get<ShaclService>()
    val populationService = InitialPopulationService(client, httpClient, provenanceService, shaclService)

    try {
        logger.info("=== EPOS Neo4j Initial Population ===")

        Config.printConfig()

        populationService.populateInitial()

        logger.info("Population completed successfully!")

    } catch (e: Exception) {
        logger.error("Error during population: {}", e.message, e)
        throw e
    } finally {
        httpClient.close()
        client.close()
        stopKoin()
    }
}

private fun startServer() {
    val port = System.getenv("PORT")?.toIntOrNull() ?: 8080
    val host = System.getenv("HOST") ?: "0.0.0.0"

    printBanner(host, port)
    Config.printConfig()

    embeddedServer(Netty, port = port, host = host, module = Application::module)
        .start(wait = true)
}

private fun printBanner(host: String, port: Int) {
    logger.info(
        "metadata_lifecycle_startup",
        entries(
            mapOf(
                "event" to "metadata_lifecycle_startup",
                "service.name" to "EPOS Metadata Lifecycle Service",
                "runtime" to "Neo4j Community | Ktor API",
                "website" to "https://www.epos-eu.org/",
                "database.uri" to Config.NEO4J_URI,
                "database.name" to Config.NEO4J_DATABASE,
                "server.url" to "http://$host:$port",
                "swagger.url" to "http://$host:$port/swagger",
            ),
        ),
    )
}

fun Application.module() {
    install(Koin) {
        slf4jLogger()
        modules(appModule)
    }

    configureLogging()
    configureContentNegotiation()
    configureCors()
    configureErrorHandling()

    val client by inject<PropertyGraphClient>()
    val resourceController by inject<ResourceController>()
    val shaclService by inject<ShaclService>()
    val provenanceService by inject<ProvenanceService>()
    val populationService by inject<InitialPopulationService>()
    val revisionImportService by inject<RevisionImportService>()
    val relationReconciliationService by inject<RelationReconciliationService>()

    runBlocking {
        shaclService.initialize()
        ensureNeo4jIndexes(client)
        warmupQueries(client)
        val resourceTypes = shaclService.getResourceTypes()
        resourceController.setResourceType(resourceTypes)
        OpenApiSpec.initializeRuntime(resourceTypes)
    }

    configureOpenApi()

    routing {
        swaggerRoutes()

        healthRoutes()

        resourceTypeRoutes(shaclService, resourceController)
        entitiesRoutes(resourceController, shaclService, client)
        resourceCrudRoutes(resourceController, shaclService)
        lifecycleRoutes(resourceController, client, provenanceService)
        batchRoutes(resourceController)

        searchRoutes(client, shaclService)
        accessRoutes(client, shaclService)

        cypherRoutes(client)
        sparqlRoutes(client)

        oaiPmhRoutes(client, shaclService)

        populationRoutes(populationService, revisionImportService, relationReconciliationService)
    }
}

private fun ensureNeo4jIndexes(client: PropertyGraphClient) {
    val statements = listOf(
        "CREATE CONSTRAINT conceptual_id_unique IF NOT EXISTS FOR (c:Conceptual) REQUIRE c.id IS UNIQUE",
        "CREATE CONSTRAINT version_id_unique IF NOT EXISTS FOR (v:Version) REQUIRE v.id IS UNIQUE",
        "CREATE INDEX version_status_idx IF NOT EXISTS FOR (v:Version) ON (v.status)",
        "CREATE INDEX version_identifier_idx IF NOT EXISTS FOR (v:Version) ON (v.identifier)",
        "CREATE INDEX conceptual_type_idx IF NOT EXISTS FOR (c:Conceptual) ON (c.type)",
        "CREATE INDEX conceptual_identifier_idx IF NOT EXISTS FOR (c:Conceptual) ON (c.identifier)",
    )

    try {
        statements.forEach { client.runWrite(it) }
        logger.info("Neo4j constraints and indexes ensured")
    } catch (e: Exception) {
        logger.warn("Failed to ensure Neo4j indexes at startup: {}", e.message)
    }
}

private fun warmupQueries(client: PropertyGraphClient) {
    val warmups = listOf(
        "MATCH (v:Version) RETURN v LIMIT 1",
        "MATCH (c:Conceptual) RETURN c LIMIT 1",
    )

    warmups.forEach { query ->
        runCatching { client.runQuery(query) }
            .onFailure { e -> logger.debug("Warmup query failed: {}", e.message) }
    }
}
