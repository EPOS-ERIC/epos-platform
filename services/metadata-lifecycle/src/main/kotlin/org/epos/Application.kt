package org.epos

import io.ktor.client.*
import io.ktor.client.engine.cio.*
import io.ktor.server.application.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.routing.*
import kotlinx.coroutines.runBlocking
import org.epos.api.controllers.ResourceController
import org.epos.di.appModule
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.epos.graphdb.services.InitialPopulationService
import org.epos.graphdb.services.ProvenanceService
import org.epos.infrastructure.ktor.plugins.*
import org.epos.infrastructure.ktor.routes.*
import org.epos.services.ShaclService
import org.koin.core.context.startKoin
import org.koin.core.context.stopKoin
import org.koin.ktor.ext.inject
import org.koin.ktor.plugin.Koin
import org.koin.logger.slf4jLogger

/**
 * EPOS GraphDB Lifecycle Management System - Application Entry Point
 *
 * Usage:
 *   java -jar app.jar                    # Start API server (default)
 *   java -jar app.jar server             # Start API server
 *   java -jar app.jar populate           # Run initial population only
 *   java -jar app.jar populate-and-serve # Run population, then start server
 *   java -jar app.jar --help             # Show help
 *
 * Environment Variables:
 *   EPOS_GRAPHDB_ENDPOINT  GraphDB server URL
 *   EPOS_REPOSITORY_ID     GraphDB repository name
 *   EPOS_INDEX_URL         URL to index.txt file
 *   EPOS_BASE_URI          Base URI for generated resources
 *   PORT                   Server port (default: 8080)
 *   HOST                   Server host (default: 0.0.0.0)
 */
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
        |   EPOS GraphDB Lifecycle Management System
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
        |  EPOS_GRAPHDB_ENDPOINT  GraphDB server URL
        |  EPOS_REPOSITORY_ID     GraphDB repository name
        |  EPOS_INDEX_URL         URL to index.txt file
        |  EPOS_BASE_URI          Base URI for generated resources
        |  PORT                   Server port (default: 8080)
        |  HOST                   Server host (default: 0.0.0.0)
        |
        |Examples:
        |  java -jar app.jar server
        |  java -jar app.jar populate
        |  EPOS_GRAPHDB_ENDPOINT=http://prod:7200 java -jar app.jar populate-and-serve
        |
    """.trimMargin()
    )
}

/**
 * Run initial population from TTL files
 */
private suspend fun runPopulation() {
    // Initialize Koin for population
    startKoin {
        slf4jLogger()
        modules(appModule)
    }

    val httpClient = org.koin.core.context.GlobalContext.get().get<HttpClient>()
    val graphDBClient = org.koin.core.context.GlobalContext.get().get<GraphDBClient>()
    val provenanceService = org.koin.core.context.GlobalContext.get().get<ProvenanceService>()
    val populationService = InitialPopulationService(graphDBClient, httpClient, provenanceService)

    try {
        println("======================================================================")
        println("   EPOS GraphDB Initial Population")
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
        graphDBClient.close()
        stopKoin()
    }
}

/**
 * Start the Ktor REST API server
 */
private fun startServer() {
    val port = System.getenv("PORT")?.toIntOrNull() ?: 8080
    val host = System.getenv("HOST") ?: "0.0.0.0"

    println("======================================================================")
    printBanner()
    println("======================================================================")
    println("   EPOS GraphDB REST API Server")
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

/**
 * Ktor application module - configures plugins and routes
 */
fun Application.module() {
    // Install Koin for dependency injection
    install(Koin) {
        slf4jLogger()
        modules(appModule)
    }

    // Install Ktor plugins
    configureLogging()
    configureContentNegotiation()
    configureCors()
    configureErrorHandling()
    configureOpenApi()

    // Inject dependencies
    val graphDBClient by inject<GraphDBClient>()
    val resourceController by inject<ResourceController>()
    val shaclService by inject<ShaclService>()
    val provenanceService by inject<ProvenanceService>()
    val populationService by inject<InitialPopulationService>()

    // Initialize SHACL service (parse resource types)
    runBlocking {
        shaclService.initialize()
        resourceController.setResourceType(shaclService.getResourceTypes())
    }

    // Configure routes
    routing {
        // Documentation
        swaggerRoutes()

        // Health
        healthRoutes()

        // API routes - Resource management
        resourceTypeRoutes(shaclService, resourceController)
        resourceCrudRoutes(resourceController, shaclService, graphDBClient)
        lifecycleRoutes(resourceController, graphDBClient, provenanceService)
        batchRoutes(resourceController)

        // Search and Discovery (new v2 API)
        searchRoutes(graphDBClient, shaclService)
        categoryRoutes(graphDBClient)
        datasetRoutes(graphDBClient, resourceController)
        distributionRoutes(graphDBClient)
        softwareRoutes(graphDBClient)

        // SPARQL endpoint
        sparqlRoutes(graphDBClient)

        // OAI-PMH endpoint for harvesting
        oaiPmhRoutes(graphDBClient, shaclService)

        // Admin routes (population, etc.)
        populationRoutes(populationService)
    }
}
