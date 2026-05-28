package org.epos.infrastructure.ktor.plugins

import io.ktor.http.ContentType
import io.ktor.http.HttpHeaders
import io.ktor.server.application.Application
import io.ktor.server.html.respondHtml
import io.ktor.server.response.header
import io.ktor.server.response.respondText
import io.ktor.server.routing.get
import io.ktor.server.routing.routing
import kotlinx.html.body
import kotlinx.html.div
import kotlinx.html.head
import kotlinx.html.id
import kotlinx.html.link
import kotlinx.html.style
import kotlinx.html.script
import kotlinx.html.title
import kotlinx.html.unsafe

fun Application.configureOpenApi() {
    routing {
        get("/openapi.runtime.yaml") {
            call.response.header(HttpHeaders.CacheControl, "no-store, no-cache, must-revalidate")
            call.respondText(OpenApiSpec.runtimeYaml, ContentType.Text.Plain)
        }

        get("/swagger") {
            call.response.header(HttpHeaders.CacheControl, "no-store, no-cache, must-revalidate")
            call.respondHtml {
                head {
                    title { +"EPOS Metadata API - Swagger UI" }
                    link(rel = "stylesheet", href = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css")
                    style {
                        unsafe {
                            +"""
                            html, body { margin: 0; padding: 0; }
                            #swagger-ui { min-height: 100vh; }
                            """.trimIndent()
                        }
                    }
                }
                body {
                    div { id = "swagger-ui" }
                    script(src = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js") {}
                    script(src = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js") {}
                    script {
                        unsafe {
                            +
                                """
                                window.onload = function() {
                                    SwaggerUIBundle({
                                        url: '/openapi.runtime.yaml',
                                        dom_id: '#swagger-ui',
                                        deepLinking: true,
                                        presets: [
                                            SwaggerUIBundle.presets.apis,
                                            SwaggerUIStandalonePreset
                                        ],
                                        layout: "StandaloneLayout"
                                    });
                                };
                                """.trimIndent()
                        }
                    }
                }
            }
        }
    }
}
