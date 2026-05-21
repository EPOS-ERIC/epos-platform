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
                    "janusgraphHost" to Config.JANUSGRAPH_HOST,
                    "janusgraphPort" to Config.JANUSGRAPH_PORT,
                    "traversalSource" to Config.JANUSGRAPH_TRAVERSAL_SOURCE
                )
            )
        )
    }
}
