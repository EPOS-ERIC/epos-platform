package org.epos.infrastructure.ktor.routes

import io.ktor.http.*
import io.ktor.server.application.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.services.ShaclService

fun Route.oaiPmhRoutes(
    client: PropertyGraphClient,
    shaclService: ShaclService
) {
    route("/oai-pmh") {
        get {
            handleOaiRequest(call, client, shaclService)
        }

        post {
            handleOaiRequest(call, client, shaclService)
        }
    }
}

private suspend fun handleOaiRequest(
    call: ApplicationCall,
    client: PropertyGraphClient,
    shaclService: ShaclService
) {
    val params =
        if (call.request.httpMethod == HttpMethod.Post) {
            runCatching { call.receiveParameters() }.getOrDefault(call.request.queryParameters)
        } else {
            call.request.queryParameters
        }

    val verb = params["verb"]
    if (verb.isNullOrBlank()) {
        call.respondText(buildOaiError("badVerb", "verb parameter is required"), ContentType.Text.Xml)
        return
    }

    val response = when (verb) {
        "Identify" -> handleIdentify()
        "ListMetadataFormats" -> handleListMetadataFormats()
        "ListSets" -> handleListSets(shaclService)
        "ListIdentifiers" -> handleListIdentifiers(client, params, shaclService)
        "ListRecords" -> handleListRecords(client, params, shaclService)
        "GetRecord" -> handleGetRecord(client, params)
        else -> {
            call.respondText(buildOaiError("badVerb", "Unknown verb: $verb"), ContentType.Text.Xml)
            return
        }
    }

    call.respondText(response, ContentType.Text.Xml)
}

private fun handleIdentify(): String {
    val now = java.time.ZonedDateTime.now().format(java.time.format.DateTimeFormatter.ISO_INSTANT)
    return buildString {
        append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
        append("<OAI-PMH xmlns=\"http://www.openarchives.org/OAI/2.0/\">\n")
        append("  <responseDate>$now</responseDate>\n")
        append("  <request verb=\"Identify\">http://localhost:8080/oai-pmh</request>\n")
        append("  <Identify>\n")
        append("    <repositoryName>EPOS Metadata Repository</repositoryName>\n")
        append("    <baseURL>http://localhost:8080/oai-pmh</baseURL>\n")
        append("    <protocolVersion>2.0</protocolVersion>\n")
        append("    <adminEmail>admin@epos-eu.org</adminEmail>\n")
        append("    <earliestDatestamp>2024-01-01T00:00:00Z</earliestDatestamp>\n")
        append("    <deletedRecord>no</deletedRecord>\n")
        append("    <granularity>YYYY-MM-DDThh:mm:ssZ</granularity>\n")
        append("  </Identify>\n")
        append("</OAI-PMH>")
    }
}

private fun handleListMetadataFormats(): String {
    val now = java.time.ZonedDateTime.now().format(java.time.format.DateTimeFormatter.ISO_INSTANT)
    return buildString {
        append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
        append("<OAI-PMH xmlns=\"http://www.openarchives.org/OAI/2.0/\">\n")
        append("  <responseDate>$now</responseDate>\n")
        append("  <request verb=\"ListMetadataFormats\">http://localhost:8080/oai-pmh</request>\n")
        append("  <ListMetadataFormats>\n")
        append("    <metadataFormat>\n")
        append("      <metadataPrefix>oai_dc</metadataPrefix>\n")
        append("      <schema>http://www.openarchives.org/OAI/2.0/oai_dc.xsd</schema>\n")
        append("      <metadataNamespace>http://www.openarchives.org/OAI/2.0/oai_dc/</metadataNamespace>\n")
        append("    </metadataFormat>\n")
    append("    <metadataFormat>\n")
    append("      <metadataPrefix>dcat</metadataPrefix>\n")
    append("      <schema>http://www.w3.org/ns/dcat</schema>\n")
    append("      <metadataNamespace>http://www.w3.org/ns/dcat#</metadataNamespace>\n")
    append("    </metadataFormat>\n")
    append("    <metadataFormat>\n")
    append("      <metadataPrefix>epos_dcat_ap</metadataPrefix>\n")
    append("      <schema>https://www.epos-eu.org/epos-dcat-ap</schema>\n")
    append("      <metadataNamespace>https://www.epos-eu.org/epos-dcat-ap#</metadataNamespace>\n")
    append("    </metadataFormat>\n")
    append("  </ListMetadataFormats>\n")
    append("</OAI-PMH>")
  }
}

private fun handleListSets(shaclService: ShaclService): String {
    val now = java.time.ZonedDateTime.now().format(java.time.format.DateTimeFormatter.ISO_INSTANT)
    val resourceTypes = shaclService.getResourceTypes()
    val setsXml = resourceTypes.joinToString("\n    ") { type ->
        "<set>\n      <setSpec>${type.name}</setSpec>\n      <setName>${type.name}</setName>\n    </set>"
    }

    return buildString {
        append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
        append("<OAI-PMH xmlns=\"http://www.openarchives.org/OAI/2.0/\">\n")
        append("  <responseDate>$now</responseDate>\n")
        append("  <request verb=\"ListSets\">http://localhost:8080/oai-pmh</request>\n")
        append("  <ListSets>\n")
        append("    $setsXml\n")
        append("  </ListSets>\n")
        append("</OAI-PMH>")
    }
}

private fun handleListIdentifiers(
    client: PropertyGraphClient,
    params: Parameters,
    shaclService: ShaclService
): String {
    val now = java.time.ZonedDateTime.now().format(java.time.format.DateTimeFormatter.ISO_INSTANT)
    val set = params["set"]

    val results = fetchPublishedRecords(client, set, shaclService)

    val identifiersXml = results.joinToString("\n    ") { record ->
        "<header identifier=\"${record.id.escapeXml()}\" datestamp=\"${record.datestamp}\" status=\"available\"/>"
    }

    val setAttr = set?.let { " set=\"$it\"" } ?: ""

    return buildString {
        append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
        append("<OAI-PMH xmlns=\"http://www.openarchives.org/OAI/2.0/\">\n")
        append("  <responseDate>$now</responseDate>\n")
        append("  <request verb=\"ListIdentifiers\"$setAttr>http://localhost:8080/oai-pmh</request>\n")
        append("  <ListIdentifiers>\n")
        append("    $identifiersXml\n")
        append("  </ListIdentifiers>\n")
        append("</OAI-PMH>")
    }
}

private fun handleListRecords(
    client: PropertyGraphClient,
    params: Parameters,
    shaclService: ShaclService
): String {
    val now = java.time.ZonedDateTime.now().format(java.time.format.DateTimeFormatter.ISO_INSTANT)
    val set = params["set"]
    val metadataPrefix = params["metadataPrefix"] ?: "oai_dc"

    val results = fetchPublishedRecords(client, set, shaclService)

    val recordsXml = results.joinToString("\n    ") { record ->
        val metadata = formatMetadata(record, metadataPrefix)
        "<record>\n      <header identifier=\"${record.id.escapeXml()}\" datestamp=\"${record.datestamp}\" status=\"available\"/>\n      <metadata>\n        $metadata\n      </metadata>\n    </record>"
    }

    val setAttr = set?.let { " set=\"$it\"" } ?: ""

    return buildString {
        append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
        append("<OAI-PMH xmlns=\"http://www.openarchives.org/OAI/2.0/\">\n")
        append("  <responseDate>$now</responseDate>\n")
        append("  <request verb=\"ListRecords\" metadataPrefix=\"$metadataPrefix\"$setAttr>http://localhost:8080/oai-pmh</request>\n")
        append("  <ListRecords>\n")
        append("    $recordsXml\n")
        append("  </ListRecords>\n")
        append("</OAI-PMH>")
    }
}

private fun handleGetRecord(
    client: PropertyGraphClient,
    params: Parameters
): String {
    val now = java.time.ZonedDateTime.now().format(java.time.format.DateTimeFormatter.ISO_INSTANT)
    val identifier = params["identifier"]
    val metadataPrefix = params["metadataPrefix"] ?: "oai_dc"

    if (identifier.isNullOrBlank()) {
        return buildOaiError("badArgument", "identifier parameter is required")
    }

    val record = fetchRecordById(client, identifier)
    if (record == null) {
        return buildOaiError("idDoesNotExist", "Record not found: $identifier")
    }

    val metadata = formatMetadata(record, metadataPrefix)

    return buildString {
        append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
        append("<OAI-PMH xmlns=\"http://www.openarchives.org/OAI/2.0/\">\n")
        append("  <responseDate>$now</responseDate>\n")
        append("  <request verb=\"GetRecord\" identifier=\"${identifier.escapeXml()}\" metadataPrefix=\"$metadataPrefix\">http://localhost:8080/oai-pmh</request>\n")
        append("  <GetRecord>\n")
        append("    <record>\n")
        append("      <header identifier=\"${record.id.escapeXml()}\" datestamp=\"${record.datestamp}\" status=\"available\"/>\n")
        append("      <metadata>\n")
        append("        $metadata\n")
        append("      </metadata>\n")
        append("    </record>\n")
        append("  </GetRecord>\n")
        append("</OAI-PMH>")
    }
}

private data class OaiRecord(
    val id: String,
    val datestamp: String,
    val title: String?,
    val description: String?,
    val type: String,
    val data: Map<String, Any?>
)

private fun fetchPublishedRecords(
    client: PropertyGraphClient,
    set: String?,
    shaclService: ShaclService
): List<OaiRecord> {
    val typeFilter: String? = if (set != null) {
        val typeInfo = shaclService.getResourceTypes().find { it.name == set }
        typeInfo?.targetClass
    } else null

    val results = client.runQuery("""
        g.V().hasLabel('Conceptual').as('c').out('CURRENT_VERSION').hasLabel('Version').has('status','published').as('v')
          .select('c','v').by(valueMap(true)).by(valueMap(true)).toList()
    """.trimIndent())
        .filter { row ->
            if (typeFilter == null || typeFilter.isBlank()) return@filter true
            val c = row["c"] as? Map<*, *> ?: return@filter false
            prop(c["type"]) == typeFilter
        }
        .take(100)

    val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    return results.mapNotNull { row ->
        val c = row["c"] as? Map<*, *> ?: return@mapNotNull null
        val id = prop(c["id"]) ?: return@mapNotNull null
        val v = row["v"] as? Map<*, *> ?: return@mapNotNull null
        val data = v.filterKeys { it is String && it !in structuralKeys }
            .mapKeys { (k, _) -> k as String }
            .mapValues { (_, vv) -> propAny(vv) }
        OaiRecord(
            id = id,
            datestamp = (prop(v["updatedAt"]) ?: "2024-01-01T00:00:00Z").take(19) + "Z",
            title = data["title"] as? String,
            description = data["description"] as? String,
            type = prop(c["type"]) ?: "Resource",
            data = data
        )
    }
}

private fun fetchRecordById(client: PropertyGraphClient, identifier: String): OaiRecord? {
    val result = client.runQuery("""
        g.V().hasLabel('Conceptual').has('id', ${gStr(identifier)}).as('c')
          .out('CURRENT_VERSION').hasLabel('Version').has('status','published').as('v')
          .select('c','v').by(valueMap(true)).by(valueMap(true)).limit(1).toList()
    """.trimIndent()).firstOrNull() ?: return null

    val c = result["c"] as? Map<*, *> ?: return null
    val id = prop(c["id"]) ?: return null
    val v = result["v"] as? Map<*, *> ?: return null
    val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    val data = v.filterKeys { it is String && it !in structuralKeys }
        .mapKeys { (k, _) -> k as String }
        .mapValues { (_, vv) -> propAny(vv) }
    return OaiRecord(
        id = id,
        datestamp = (prop(v["updatedAt"]) ?: "2024-01-01T00:00:00Z").take(19) + "Z",
        title = data["title"] as? String,
        description = data["description"] as? String,
        type = prop(c["type"]) ?: "Resource",
        data = data
    )
}

private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"
private fun prop(value: Any?): String? = when (value) {
    is List<*> -> value.firstOrNull()?.toString()
    else -> value?.toString()
}
private fun propAny(value: Any?): Any? =
    when (value) {
        is List<*> -> if (value.size == 1) value.firstOrNull() else value
        else -> value
    }

private fun formatMetadata(record: OaiRecord, metadataPrefix: String): String {
    return when (metadataPrefix) {
        "oai_dc" -> formatOaiDc(record)
        "dcat" -> formatDcat(record)
        else -> formatOaiDc(record)
    }
}

private fun formatOaiDc(record: OaiRecord): String {
    val title = record.title?.let { "<dc:title>${it.escapeXml()}</dc:title>" } ?: ""
    val description = record.description?.let { "<dc:description>${it.escapeXml()}</dc:description>" } ?: ""
    val identifier = "<dc:identifier>${record.id.escapeXml()}</dc:identifier>"
    val type = "<dc:type>${record.type.escapeXml()}</dc:type>"

    return "<oai_dc:dc xmlns:oai_dc=\"http://www.openarchives.org/OAI/2.0/oai_dc/\" xmlns:dc=\"http://purl.org/dc/elements/1.1/\">\n" +
        "        $title\n" +
        "        $description\n" +
        "        $identifier\n" +
        "        $type\n" +
        "      </oai_dc:dc>"
}

private fun formatDcat(record: OaiRecord): String {
    return "<dcat:Catalog xmlns:dcat=\"http://www.w3.org/ns/dcat#\" xmlns:dcterms=\"http://purl.org/dc/terms/\">\n" +
        "        <dcterms:title>${record.title?.escapeXml() ?: record.id.escapeXml()}</dcterms:title>\n" +
        "        <dcterms:identifier>${record.id.escapeXml()}</dcterms:identifier>\n" +
        "      </dcat:Catalog>"
}

private fun String.escapeXml(): String =
    this.replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace("\"", "&quot;")
        .replace("'", "&apos;")

private fun buildOaiError(code: String, message: String): String {
    val now = java.time.ZonedDateTime.now().format(java.time.format.DateTimeFormatter.ISO_INSTANT)
    return buildString {
        append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
        append("<OAI-PMH xmlns=\"http://www.openarchives.org/OAI/2.0/\">\n")
        append("  <responseDate>$now</responseDate>\n")
        append("  <error code=\"$code\">$message</error>\n")
        append("</OAI-PMH>")
    }
}
