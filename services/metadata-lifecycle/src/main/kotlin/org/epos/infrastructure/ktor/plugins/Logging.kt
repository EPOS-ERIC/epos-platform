package org.epos.infrastructure.ktor.plugins

import io.ktor.server.application.*
import io.ktor.server.request.*
import io.ktor.server.plugins.calllogging.*
import org.slf4j.event.Level

/**
 * Configures request/response logging.
 */
fun Application.configureLogging() {
    install(CallLogging) {
        level = Level.INFO
        format { call ->
            val status = call.response.status()?.value ?: 0
            val method = call.request.httpMethod.value
            val path = call.request.path()
            "http_request method=$method path=$path status=$status"
        }
    }
}
