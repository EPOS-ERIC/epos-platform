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
import kotlinx.html.script
import kotlinx.html.title
import kotlinx.html.unsafe
import org.epos.services.ShaclService

fun Application.configureOpenApi(shaclService: ShaclService) {
    routing {
        get("/openapi.yaml") {
            call.response.header(HttpHeaders.CacheControl, "no-store, no-cache, must-revalidate")
            call.respondText(OpenApiSpec.yaml, ContentType.Text.Plain)
        }

        get("/openapi.runtime.yaml") {
            call.response.header(HttpHeaders.CacheControl, "no-store, no-cache, must-revalidate")
            call.respondText(OpenApiSpec.runtimeYaml, ContentType.Text.Plain)
        }

        get("/swagger") {
            call.response.header(HttpHeaders.CacheControl, "no-store, no-cache, must-revalidate")
            val renderedAt = System.currentTimeMillis()
            val runtimeSpecFingerprint = OpenApiSpec.runtimeYaml.hashCode()
            call.respondHtml {
                head {
                    title { +"EPOS Metadata API - Swagger UI" }
                    link(rel = "stylesheet", href = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css")
                }
                body {
                    div {
                        +"Swagger source: /openapi.runtime.yaml | renderedAt=$renderedAt | specFingerprint=$runtimeSpecFingerprint"
                    }
                    div { id = "swagger-ui" }
                    script(src = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js") {}
                    script {
                        unsafe {
                            +
                                """
                                window.onload = function() {
                                    const ui = SwaggerUIBundle({
                                        url: '/openapi.runtime.yaml',
                                        dom_id: '#swagger-ui',
                                        presets: [
                                            SwaggerUIBundle.presets.apis,
                                            SwaggerUIBundle.SwaggerUIStandalonePreset
                                        ],
                                        layout: "BaseLayout"
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
