package org.epos.di

import io.ktor.client.HttpClient
import io.ktor.client.engine.cio.CIO
import org.epos.api.controllers.ResourceController
import org.epos.api.shacl.ShaclParser
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.services.EntityLifecycleService
import org.epos.propertygraph.services.ProvenanceService
import org.epos.propertygraph.services.population.InitialPopulationService
import org.epos.propertygraph.services.population.RelationReconciliationService
import org.epos.propertygraph.services.population.RevisionImportService
import org.epos.services.ShaclService
import org.koin.dsl.module

val appModule =
    module {
        single { HttpClient(CIO) }

        single { PropertyGraphClient() }

        single { ProvenanceService(get()) }
        single { EntityLifecycleService(get(), get()) }
        single { InitialPopulationService(get(), get(), get(), get()) }
        single { RevisionImportService(get(), get(), get(), get()) }
        single { RelationReconciliationService(get()) }

        single { ShaclParser(get()) }
        single { ShaclService(get()) }

        single { ResourceController(get(), get(), get()) }
    }

val testModule =
    module {
    }
