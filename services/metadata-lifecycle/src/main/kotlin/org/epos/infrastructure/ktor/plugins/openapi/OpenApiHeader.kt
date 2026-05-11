package org.epos.infrastructure.ktor.plugins

object OpenApiHeader {
    val yaml =
        """
openapi: 3.1.0
info:
  title: EPOS API
  version: 2.0.0
  description: |
    OpenAPI specification for the EPOS catalogue API.

    ## Design Principles
    - Search returns heterogeneous, typed resource items in a flat list.
    - Clients navigate by following `href` links; they never construct URLs themselves.
    - All cross-entity references carry an `href` pointing to the typed detail endpoint.

    ## Lifecycle States
    - **DRAFT** - Resource is being edited
    - **SUBMITTED** - Resource submitted for review
    - **PUBLISHED** - Resource is publicly visible
    - **ARCHIVED** - Outdated versions kept for historical purposes

    ## Authentication
    Mutating operations require both headers with absolute URIs:
    - `X-Agent` identifies the acting agent.
    - `X-Group` identifies the acting group.
    Example:
    - `X-Agent: mailto:user@example.org`
    - `X-Group: https://epos-eu.org/group/ingestion-team`
  termsOfService: https://www.epos-eu.org/sites/default/files/2026-02/Terms_and_ConditionsJan26_1.pdf
  contact:
    name: EPOS Development Team
    email: support@epos-eu.org
    url: https://epos-eu.org
  license:
    name: AGPL-3.0
    url: https://www.gnu.org/licenses/agpl-3.0.en.html

servers:
  - url: http://localhost:8080
    description: Development server

tags:
  - name: Search
    description: Search across all resource types
  - name: Access
    description: Unified shape access for list and detail retrieval
  - name: Datasets
    description: Dataset resources
  - name: Distributions
    description: Distribution resources
  - name: Software Applications
    description: Software application resources
  - name: Software Source Codes
    description: Software source code resources
  - name: Categories
    description: Category taxonomy resources
  - name: System
    description: System information and health checks
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

paths:
        """.trimIndent()
}
