package org.epos.di

import io.ktor.client.*
import io.ktor.client.engine.cio.*
import org.epos.api.controllers.ResourceController
import org.epos.api.shacl.ShaclParser
import org.epos.graphdb.client.GraphDBClient
import org.epos.graphdb.services.EntityLifecycleService
import org.epos.graphdb.services.InitialPopulationService
import org.epos.graphdb.services.ProvenanceService
import org.epos.services.EdmMappingService
import org.epos.services.ShaclService
import org.koin.dsl.module

/**
 * Main Koin dependency injection module.
 * Defines all application dependencies and their lifecycle.
 */
val appModule = module {
    // HTTP Client - singleton, shared across services
    single { HttpClient(CIO) }

    // GraphDB Client - singleton
    single { GraphDBClient(get()) }

    // Services - singletons
    single { ProvenanceService(get()) }
    single { EntityLifecycleService(get(), get()) }
    single { InitialPopulationService(get(), get(), get()) }

    // SHACL Service - manages resource type parsing
    single { ShaclParser(get()) }
    single { ShaclService(get(), get()) }
    
    // EDM Mapping Service - provides EPOS Data Model property mappings
    single { EdmMappingService(get()) }

    // Controllers - singletons (they're stateless)
    single { ResourceController(get(), get(), get()) }
}

/**
 * Module for test environment with mock-friendly configuration.
 * Can be used in integration tests to override specific dependencies.
 */
val testModule = module {
    // Override specific dependencies for testing if needed
}
