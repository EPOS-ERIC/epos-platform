package org.epos.infrastructure.ktor.routes

import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.epos.graphdb.Config

/**
 * Health check and status routes.
 */
fun Route.healthRoutes() {
    get("/health") {
        call.respond(
            mapOf(
                "status" to "ok",
                "timestamp" to System.currentTimeMillis(),
                "config" to mapOf(
                    "graphdbEndpoint" to Config.GRAPHDB_ENDPOINT,
                    "repositoryId" to Config.REPOSITORY_ID
                )
            )
        )
    }
}
