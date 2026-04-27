package org.epos.infrastructure.ktor.routes

import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.epos.api.controllers.ResourceController
import org.epos.api.models.BatchOperationRequest

/**
 * Batch operation routes (bulk submit, publish).
 */
fun Route.batchRoutes(resourceController: ResourceController) {
    route("/api/v2/batch") {
        post("/submit") {
            val request = call.receive<BatchOperationRequest>()
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.submitEntityBatch(request.entityUris, actorHeaders.agent, actorHeaders.group)
            call.respond(response)
        }

        post("/publish") {
            val request = call.receive<BatchOperationRequest>()
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.publishEntityBatch(request.entityUris, actorHeaders.agent, actorHeaders.group)
            call.respond(response)
        }
    }
}
