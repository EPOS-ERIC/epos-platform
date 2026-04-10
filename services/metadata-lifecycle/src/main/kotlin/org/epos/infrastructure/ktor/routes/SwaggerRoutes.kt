package org.epos.infrastructure.ktor.routes

import io.ktor.server.response.*
import io.ktor.server.routing.*

/**
 * Routes for Swagger UI redirect.
 * The actual Swagger UI is served by the SwaggerUI plugin at /swagger
 */
fun Route.swaggerRoutes() {
    // Redirect old path to new swagger location
    get("/metadata-lifecycle-service") {
        call.respondRedirect("/swagger")
    }
}
