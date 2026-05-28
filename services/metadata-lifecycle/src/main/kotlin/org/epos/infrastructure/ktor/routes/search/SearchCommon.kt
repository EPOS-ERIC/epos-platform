package org.epos.infrastructure.ktor.routes

internal fun extractTypeName(typeUri: String): String = typeUri.substringAfterLast("/").substringAfterLast("#")

