package org.epos.infrastructure.ktor.routes

import io.ktor.http.HttpStatusCode
import io.ktor.server.application.ApplicationCall
import io.ktor.server.request.header
import io.ktor.server.response.respond
import org.epos.api.models.ErrorResponse
import org.epos.propertygraph.utils.UriUtils

data class ActorHeaders(
    val agent: String,
    val group: String,
)

suspend fun ApplicationCall.requireActorHeaders(): ActorHeaders? {
    val agent = request.header("X-Agent")?.trim()
    if (agent.isNullOrBlank()) {
        respond(HttpStatusCode.BadRequest, ErrorResponse("BadRequest", "Missing required header: X-Agent"))
        return null
    }

    val group = request.header("X-Group")?.trim()
    if (group.isNullOrBlank()) {
        respond(HttpStatusCode.BadRequest, ErrorResponse("BadRequest", "Missing required header: X-Group"))
        return null
    }

    if (!UriUtils.isAbsolute(agent)) {
        respond(HttpStatusCode.BadRequest, ErrorResponse("BadRequest", "Header X-Agent must be an absolute URI"))
        return null
    }

    if (!UriUtils.isAbsolute(group)) {
        respond(HttpStatusCode.BadRequest, ErrorResponse("BadRequest", "Header X-Group must be an absolute URI"))
        return null
    }

    return ActorHeaders(agent = agent, group = group)
}
