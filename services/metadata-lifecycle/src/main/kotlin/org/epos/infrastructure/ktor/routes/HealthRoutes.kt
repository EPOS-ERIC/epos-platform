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
                    "neo4jUri" to Config.NEO4J_URI,
                    "neo4jDatabase" to Config.NEO4J_DATABASE
                )
            )
        )
    }
}
