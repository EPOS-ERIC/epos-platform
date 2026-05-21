package org.epos

import io.ktor.client.*
import io.ktor.server.application.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.routing.*
import kotlinx.coroutines.runBlocking
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
            println("Starting EPOS API Server...")
            startServer()
        }
        "populate", "init" -> {
            println("Running Initial Population...")
            runPopulation()
        }
        "populate-and-serve", "init-and-serve", "all" -> {
            println("Running Initial Population...")
            runPopulation()
            println("\nStarting EPOS API Server...")
            startServer()
        }
        "--help", "-h", "help" -> {
            printHelp()
        }
        else -> {
            println("Unknown mode: $mode")
            printHelp()
        }
    }
}

private fun printHelp() {
    println(
        """
        |=================================================
        |   EPOS JanusGraph Metadata Lifecycle Management System
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
        |  EPOS_JANUSGRAPH_HOST   JanusGraph host
        |  EPOS_JANUSGRAPH_PORT   JanusGraph Gremlin port
        |  EPOS_JANUSGRAPH_TRAVERSAL_SOURCE  Gremlin traversal source
        |  EPOS_JANUSGRAPH_USERNAME JanusGraph username
        |  EPOS_JANUSGRAPH_PASSWORD JanusGraph password
        |  EPOS_INDEX_URL       URL to index.txt file
        |  EPOS_BASE_URI        Base URI for generated resources
        |  PORT                   Server port (default: 8080)
        |  HOST                   Server host (default: 0.0.0.0)
        |
        |Examples:
        |  java -jar app.jar server
        |  java -jar app.jar populate
        |  EPOS_JANUSGRAPH_HOST=prod java -jar app.jar populate-and-serve
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
        println("======================================================================")
        println("   EPOS JanusGraph Initial Population")
        println("======================================================================\n")

        Config.printConfig()
        println()

        populationService.populateInitial()

        println("\nPopulation completed successfully!")

    } catch (e: Exception) {
        println("\nError during population: ${e.message}")
        e.printStackTrace()
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

    println("======================================================================")
    printBanner()
    println("======================================================================")
    println("   EPOS JanusGraph REST API Server")
    println("======================================================================")
    Config.printConfig()
    println("Server: http://$host:$port")
    println("Swagger UI: http://$host:$port/swagger")
    println("======================================================================\n")

    embeddedServer(Netty, port = port, host = host, module = Application::module)
        .start(wait = true)
}

private fun printBanner() {
    println("""
        |EPOS Metadata Lifecycle Service
        |================================
    """.trimMargin())
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
        ensureJanusGraphIndexes(client)
        warmupQueries(client)
        val resourceTypes = shaclService.getResourceTypes()
        resourceController.setResourceType(resourceTypes)
        OpenApiSpec.initializeRuntime(resourceTypes)
    }

    configureOpenApi(shaclService)

    routing {
        swaggerRoutes()

        healthRoutes()

        resourceTypeRoutes(shaclService, resourceController)
        entitiesRoutes(resourceController, shaclService, client)
        resourceCrudRoutes(resourceController, shaclService, client)
        lifecycleRoutes(resourceController, client, provenanceService)
        batchRoutes(resourceController)

        searchRoutes(client, shaclService)
        accessRoutes(client, shaclService)

        datasetRoutes(client, resourceController)
        distributionRoutes(client)
        softwareRoutes(client)
        categoryRoutes(client)

        cypherRoutes(client)

        oaiPmhRoutes(client, shaclService)

        populationRoutes(populationService, revisionImportService, relationReconciliationService)
    }
}

private fun ensureJanusGraphIndexes(client: PropertyGraphClient) {
    val script =
        """
        mgmt = graph.openManagement()
        changed = []
        idKey = mgmt.getPropertyKey('id') ?: mgmt.makePropertyKey('id').dataType(String.class).cardinality(org.janusgraph.core.Cardinality.SINGLE).make()
        statusKey = mgmt.getPropertyKey('status') ?: mgmt.makePropertyKey('status').dataType(String.class).cardinality(org.janusgraph.core.Cardinality.SINGLE).make()
        identifierKey = mgmt.getPropertyKey('identifier') ?: mgmt.makePropertyKey('identifier').dataType(String.class).cardinality(org.janusgraph.core.Cardinality.SINGLE).make()
        typeKey = mgmt.getPropertyKey('type') ?: mgmt.makePropertyKey('type').dataType(String.class).cardinality(org.janusgraph.core.Cardinality.SINGLE).make()
        versionLabel = mgmt.getVertexLabel('Version') ?: mgmt.makeVertexLabel('Version').make()

        if (versionLabel != null && mgmt.getGraphIndex('idxVersionId') == null) {
            mgmt.buildIndex('idxVersionId', org.apache.tinkerpop.gremlin.structure.Vertex.class)
                .addKey(idKey)
                .indexOnly(versionLabel)
                .buildCompositeIndex()
            changed.add('idxVersionId')
        }

        if (versionLabel != null && mgmt.getGraphIndex('idxVersionStatus') == null) {
            mgmt.buildIndex('idxVersionStatus', org.apache.tinkerpop.gremlin.structure.Vertex.class)
                .addKey(statusKey)
                .indexOnly(versionLabel)
                .buildCompositeIndex()
            changed.add('idxVersionStatus')
        }

        if (versionLabel != null && mgmt.getGraphIndex('idxVersionIdentifier') == null) {
            mgmt.buildIndex('idxVersionIdentifier', org.apache.tinkerpop.gremlin.structure.Vertex.class)
                .addKey(identifierKey)
                .indexOnly(versionLabel)
                .buildCompositeIndex()
            changed.add('idxVersionIdentifier')
        }

        conceptualLabel = mgmt.getVertexLabel('Conceptual') ?: mgmt.makeVertexLabel('Conceptual').make()
        if (conceptualLabel != null && mgmt.getGraphIndex('idxConceptualId') == null) {
            mgmt.buildIndex('idxConceptualId', org.apache.tinkerpop.gremlin.structure.Vertex.class)
                .addKey(idKey)
                .indexOnly(conceptualLabel)
                .buildCompositeIndex()
            changed.add('idxConceptualId')
        }

        if (conceptualLabel != null && mgmt.getGraphIndex('idxConceptualType') == null) {
            mgmt.buildIndex('idxConceptualType', org.apache.tinkerpop.gremlin.structure.Vertex.class)
                .addKey(typeKey)
                .indexOnly(conceptualLabel)
                .buildCompositeIndex()
            changed.add('idxConceptualType')
        }

        if (conceptualLabel != null && mgmt.getGraphIndex('idxConceptualIdentifier') == null) {
            mgmt.buildIndex('idxConceptualIdentifier', org.apache.tinkerpop.gremlin.structure.Vertex.class)
                .addKey(identifierKey)
                .indexOnly(conceptualLabel)
                .buildCompositeIndex()
            changed.add('idxConceptualIdentifier')
        }

        mgmt.commit()
        changed
        """.trimIndent()

    try {
        val created = client.runQuery(script).mapNotNull { it["value"]?.toString() }
        if (created.isNotEmpty()) {
            logger.info("JanusGraph indexes created: {}", created.joinToString(", "))
            logger.info("Run JanusGraph reindex for existing data to fully eliminate full-scan warnings")
        } else {
            logger.info("JanusGraph indexes already present")
        }
    } catch (e: Exception) {
        logger.warn("Failed to ensure JanusGraph indexes at startup: {}", e.message)
    }
}

private fun warmupQueries(client: PropertyGraphClient) {
    val warmups = listOf(
        "g.V().hasLabel('Version').limit(1).count().next()",
        "g.V().hasLabel('Conceptual').limit(1).count().next()",
        "g.V().hasLabel('Conceptual').has('type','dataset').out('CURRENT_VERSION').hasLabel('Version').limit(1).count().next()",
    )

    warmups.forEach { query ->
        runCatching { client.runQuery(query) }
            .onFailure { e -> logger.debug("Warmup query failed: {}", e.message) }
    }
}
