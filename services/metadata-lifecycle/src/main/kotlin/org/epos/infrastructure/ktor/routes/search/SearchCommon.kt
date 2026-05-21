package org.epos.infrastructure.ktor.routes

internal fun extractTypeName(typeUri: String): String = typeUri.substringAfterLast("/").substringAfterLast("#")

internal fun buildSpatialFilter(
    bbox: String?,
    geo: String?,
    geoRelation: String,
    lat: Double?,
    lon: Double?,
    radius: Double?,
): String? {
    val geometryExtraction =
        """
        {
            ?s dcterms:spatial ?spatialEntity .
            ?spatialEntity geo:asWKT ?entityGeom .
        } UNION {
            ?s dcterms:spatial ?spatialEntity .
            ?spatialEntity <http://www.w3.org/ns/locn#geometry> ?geomNode .
            ?geomNode geo:asWKT ?entityGeom .
        } UNION {
            ?s geo:hasGeometry ?geomNode .
            ?geomNode geo:asWKT ?entityGeom .
        } UNION {
            ?s geo:asWKT ?entityGeom .
        } UNION {
            ?s dcat:bbox ?entityGeom .
            FILTER(DATATYPE(?entityGeom) = geo:wktLiteral)
        }
        """.trimIndent()

    return when {
        !geo.isNullOrBlank() -> {
            val spatialFunction = getSpatialFunction(geoRelation)
            """
            $geometryExtraction
            FILTER($spatialFunction(?entityGeom, "$geo"^^geo:wktLiteral))
            """.trimIndent()
        }

        !bbox.isNullOrBlank() -> {
            val coords = bbox.split(",").mapNotNull { it.trim().toDoubleOrNull() }
            if (coords.size == 4) {
                val (minLon, minLat, maxLon, maxLat) = coords
                val bboxWkt = "POLYGON(($minLon $minLat, $maxLon $minLat, $maxLon $maxLat, $minLon $maxLat, $minLon $minLat))"
                val spatialFunction = getSpatialFunction(geoRelation)
                """
                $geometryExtraction
                FILTER($spatialFunction(?entityGeom, "$bboxWkt"^^geo:wktLiteral))
                """.trimIndent()
            } else {
                null
            }
        }

        lat != null && lon != null && radius != null -> {
            val pointWkt = "POINT($lon $lat)"
            val radiusMeters = radius * 1000
            """
            $geometryExtraction
            FILTER(geof:distance(?entityGeom, "$pointWkt"^^geo:wktLiteral, uom:metre) <= $radiusMeters)
            """.trimIndent()
        }

        lat != null && lon != null -> {
            val pointWkt = "POINT($lon $lat)"
            val defaultRadiusMeters = 50000.0
            """
            $geometryExtraction
            FILTER(geof:distance(?entityGeom, "$pointWkt"^^geo:wktLiteral, uom:metre) <= $defaultRadiusMeters)
            """.trimIndent()
        }

        else -> {
            null
        }
    }
}

internal fun getSpatialFunction(relation: String): String =
    when (relation.lowercase()) {
        "within" -> "geof:sfWithin"
        "intersects" -> "geof:sfIntersects"
        "contains" -> "geof:sfContains"
        "overlaps" -> "geof:sfOverlaps"
        "touches" -> "geof:sfTouches"
        "disjoint" -> "geof:sfDisjoint"
        "crosses" -> "geof:sfCrosses"
        "equals" -> "geof:sfEquals"
        else -> "geof:sfIntersects"
    }
