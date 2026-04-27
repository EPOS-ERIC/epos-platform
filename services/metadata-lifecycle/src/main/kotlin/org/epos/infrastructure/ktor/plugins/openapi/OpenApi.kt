package org.epos.infrastructure.ktor.plugins

object OpenApiSpec {
    val yaml: String by lazy {
        listOf(
            OpenApiHeader.yaml,
            OpenApiPaths.System.yaml,
            OpenApiPaths.Search.yaml,
            OpenApiPaths.Categories.yaml,
            OpenApiPaths.Details.yaml,
            OpenApiPaths.Sparql.yaml,
            OpenApiPaths.OaiPmh.yaml,
            OpenApiPaths.Resources.yaml,
            OpenApiPaths.Lifecycle.yaml,
            OpenApiPaths.Versioning.yaml,
            OpenApiPaths.Batch.yaml,
            OpenApiPaths.Admin.yaml,
            OpenApiComponents.yaml,
        ).joinToString("\n")
    }
}

object OpenApiPaths {
    object System {
        val yaml = OpenApiSystemPaths.yaml
    }

    object Search {
        val yaml = OpenApiSearchPaths.yaml
    }

    object Categories {
        val yaml = OpenApiCategoryPaths.yaml
    }

    object Details {
        val yaml = OpenApiDetailsPaths.yaml
    }

    object Sparql {
        val yaml = OpenApiSparqlPaths.yaml
    }

    object OaiPmh {
        val yaml = OpenApiOaiPmhPaths.yaml
    }

    object Resources {
        val yaml = OpenApiResourcePaths.yaml
    }

    object Lifecycle {
        val yaml = OpenApiLifecyclePaths.yaml
    }

    object Versioning {
        val yaml = OpenApiVersioningPaths.yaml
    }

    object Batch {
        val yaml = OpenApiBatchPaths.yaml
    }

    object Admin {
        val yaml = OpenApiAdminPaths.yaml
    }
}
