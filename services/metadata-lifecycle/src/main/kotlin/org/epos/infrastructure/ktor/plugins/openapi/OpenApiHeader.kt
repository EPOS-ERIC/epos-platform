package org.epos.infrastructure.ktor.plugins

object OpenApiHeader {
    val yaml = """
openapi: 3.1.0
info:
  title: EPOS Metadata Lifecycle API
  description: |
    REST API to manage the lifecycle of RDF metadata resources in GraphDB.

    ## Lifecycle Workflow
    1. **DRAFT** - Create a new draft of a resource
    2. **SUBMITTED** - Submit the draft for review
    3. **PUBLISHED** - Publish the resource (automatically archives the previous version)
    4. **ARCHIVED** - Outdated versions kept for historical purposes

    ## Authentication
    Use the `X-Agent` header with an absolute URI to identify the agent performing the operation.
    Example: `X-Agent: mailto:user@example.org`
  version: 2.2.0
  contact:
    name: EPOS Development Team
    url: https://epos-eu.org

servers:
  - url: http://localhost:8080
    description: Development server

tags:
  - name: System
    description: System information and health checks
  - name: Search
    description: Search and discovery endpoints
  - name: SPARQL
    description: Direct SPARQL query access
  - name: OAI-PMH
    description: OAI-PMH 2.0 metadata harvesting
  - name: Resources
    description: CRUD operations on metadata resources
  - name: Lifecycle
    description: Lifecycle management (submit, publish)
  - name: Versioning
    description: Resource history and versions
  - name: Batch
    description: Batch submit and publish multiple entities
  - name: Admin
    description: Administrative operations (population, maintenance)

paths:""".trimIndent()
}
