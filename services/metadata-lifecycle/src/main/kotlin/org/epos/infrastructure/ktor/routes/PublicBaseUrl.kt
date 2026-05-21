package org.epos.infrastructure.ktor.routes

import io.ktor.server.application.ApplicationCall
import io.ktor.server.plugins.origin
import org.epos.propertygraph.Config

fun ApplicationCall.resolvePublicBaseUrl(): String {
    val configured =
        Config.server.publicBaseUrl
            ?.trim()
            ?.takeIf { it.isNotBlank() }
    if (configured != null) return configured.removeSuffix("/")

    val forwardedProto =
        request.headers["X-Forwarded-Proto"]
            ?.split(",")
            ?.firstOrNull()
            ?.trim()
    val forwardedHost =
        request.headers["X-Forwarded-Host"]
            ?.split(",")
            ?.firstOrNull()
            ?.trim()
    val forwardedPort =
        request.headers["X-Forwarded-Port"]
            ?.split(",")
            ?.firstOrNull()
            ?.trim()
            ?.toIntOrNull()

    val origin = request.origin
    val scheme = forwardedProto ?: origin.scheme
    val host = forwardedHost ?: origin.serverHost
    val port = forwardedPort ?: origin.serverPort

    val includePort = !((scheme == "http" && port == 80) || (scheme == "https" && port == 443))
    val portPart = if (includePort) ":$port" else ""
    return "$scheme://$host$portPart"
}
