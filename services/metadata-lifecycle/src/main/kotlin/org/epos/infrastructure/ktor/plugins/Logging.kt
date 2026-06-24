package org.epos.infrastructure.ktor.plugins

import io.ktor.server.application.Application
import io.ktor.server.application.install
import io.ktor.server.plugins.calllogging.CallLogging
import io.ktor.server.request.httpMethod
import io.ktor.server.request.path
import org.slf4j.event.Level

/**
 * Configures request/response logging.
 */
fun Application.configureLogging() {
    install(CallLogging) {
        level = Level.INFO
        filter { call ->
            val method = call.request.httpMethod.value
            val path = call.request.path()
            !(method == "GET" && path == "/health")
        }
        format { call ->
            val status = call.response.status()?.value ?: 0
            val method = call.request.httpMethod.value
            val path = call.request.path()
            "http_request method=$method path=$path status=$status"
        }
    }
}
