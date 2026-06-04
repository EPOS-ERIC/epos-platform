package org.epos.infrastructure.ktor.plugins

import io.ktor.server.application.Application
import io.ktor.server.application.install
import io.ktor.server.plugins.calllogging.CallLogging
import org.slf4j.event.Level

/**
 * Configures request/response logging.
 */
fun Application.configureLogging() {
    install(CallLogging) {
        level = Level.INFO
    }
}
