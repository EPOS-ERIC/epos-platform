package org.epos.infrastructure.ktor.routes

import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.epos.propertygraph.Config

fun Route.healthRoutes() {
    get("/health") {
        call.respond(
            mapOf(
                "status" to "ok",
                "timestamp" to System.currentTimeMillis(),
                "config" to mapOf(
                    "neo4jUri" to (System.getenv("EPOS_NEO4J_URI") ?: Config.neo4j.uri),
                    "neo4jDatabase" to (System.getenv("EPOS_NEO4J_DATABASE") ?: "neo4j")
                )
            )
        )
    }
}
