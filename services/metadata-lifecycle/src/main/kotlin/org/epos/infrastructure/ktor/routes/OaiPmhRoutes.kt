package org.epos.infrastructure.ktor.routes

import io.ktor.http.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.epos.services.ShaclService
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter

/**
 * OAI-PMH 2.0 compliant endpoint for metadata harvesting.
 * 
 * Supported verbs:
 * - Identify: Information about the repository
 * - ListMetadataFormats: Available metadata formats
 * - ListSets: Available sets (resource types)
 * - ListIdentifiers: List record identifiers with optional filtering
 * - ListRecords: List full records with optional filtering
 * - GetRecord: Get a single record by identifier
 * 
 * Supported metadata formats:
 * - oai_dc: Dublin Core (required by OAI-PMH)
 * - dcat: DCAT-AP RDF/XML
 * - epos_dcat_ap: EPOS-DCAT-AP Turtle
 */
fun Route.oaiPmhRoutes(graphDBClient: GraphDBClient, shaclService: ShaclService) {
    route("/oai-pmh") {
        
        get {
            handleOaiRequest(call, graphDBClient, shaclService)
        }
        
        post {
            handleOaiRequest(call, graphDBClient, shaclService)
        }
    }
}

private suspend fun handleOaiRequest(
    call: io.ktor.server.application.ApplicationCall,
    graphDBClient: GraphDBClient,
    shaclService: ShaclService
) {
    val params =
        if (call.request.httpMethod == HttpMethod.Post) {
            val formParameters =
                runCatching { call.receiveParameters() }
                    .getOrElse { Parameters.Empty }
            Parameters.build {
                call.request.queryParameters.forEach { key, values ->
                    values.forEach { append(key, it) }
                }
                formParameters.forEach { key, values ->
                    values.forEach { append(key, it) }
                }
            }
        } else {
            call.request.queryParameters
        }
    val verb = params["verb"]
    
    val responseDate = ZonedDateTime.now().format(DateTimeFormatter.ISO_INSTANT)
    val baseUrl = "${Config.BASE_URI}/oai-pmh"
    
    when (verb) {
        "Identify" -> {
            val xml = buildIdentifyResponse(responseDate, baseUrl)
            call.respondText(xml, ContentType.Text.Xml)
        }
        "ListMetadataFormats" -> {
            val identifier = params["identifier"]
            val xml = buildListMetadataFormatsResponse(responseDate, baseUrl, identifier)
            call.respondText(xml, ContentType.Text.Xml)
        }
        "ListSets" -> {
            val xml = buildListSetsResponse(responseDate, baseUrl, shaclService)
            call.respondText(xml, ContentType.Text.Xml)
        }
        "ListIdentifiers" -> {
            val metadataPrefix = params["metadataPrefix"]
            if (metadataPrefix == null) {
                call.respondText(
                    buildErrorResponse(responseDate, baseUrl, "badArgument", "Missing required argument: metadataPrefix"),
                    ContentType.Text.Xml
                )
                return
            }
            if (!isValidMetadataPrefix(metadataPrefix)) {
                call.respondText(
                    buildErrorResponse(responseDate, baseUrl, "cannotDisseminateFormat", "Unknown metadata format: $metadataPrefix"),
                    ContentType.Text.Xml
                )
                return
            }
            val set = params["set"]
            val from = params["from"]
            val until = params["until"]
            val resumptionToken = params["resumptionToken"]
            
            val xml = buildListIdentifiersResponse(
                responseDate, baseUrl, graphDBClient, 
                metadataPrefix, set, from, until, resumptionToken
            )
            call.respondText(xml, ContentType.Text.Xml)
        }
        "ListRecords" -> {
            val metadataPrefix = params["metadataPrefix"]
            if (metadataPrefix == null) {
                call.respondText(
                    buildErrorResponse(responseDate, baseUrl, "badArgument", "Missing required argument: metadataPrefix"),
                    ContentType.Text.Xml
                )
                return
            }
            if (!isValidMetadataPrefix(metadataPrefix)) {
                call.respondText(
                    buildErrorResponse(responseDate, baseUrl, "cannotDisseminateFormat", "Unknown metadata format: $metadataPrefix"),
                    ContentType.Text.Xml
                )
                return
            }
            val set = params["set"]
            val from = params["from"]
            val until = params["until"]
            val resumptionToken = params["resumptionToken"]
            
            val xml = buildListRecordsResponse(
                responseDate, baseUrl, graphDBClient,
                metadataPrefix, set, from, until, resumptionToken
            )
            call.respondText(xml, ContentType.Text.Xml)
        }
        "GetRecord" -> {
            val identifier = params["identifier"]
            val metadataPrefix = params["metadataPrefix"]
            
            if (identifier == null || metadataPrefix == null) {
                call.respondText(
                    buildErrorResponse(responseDate, baseUrl, "badArgument", "Missing required arguments: identifier and metadataPrefix"),
                    ContentType.Text.Xml
                )
                return
            }
            if (!isValidMetadataPrefix(metadataPrefix)) {
                call.respondText(
                    buildErrorResponse(responseDate, baseUrl, "cannotDisseminateFormat", "Unknown metadata format: $metadataPrefix"),
                    ContentType.Text.Xml
                )
                return
            }
            
            val xml = buildGetRecordResponse(
                responseDate, baseUrl, graphDBClient,
                identifier, metadataPrefix
            )
            call.respondText(xml, ContentType.Text.Xml)
        }
        null -> {
            call.respondText(
                buildErrorResponse(responseDate, baseUrl, "badVerb", "Missing verb argument"),
                ContentType.Text.Xml
            )
        }
        else -> {
            call.respondText(
                buildErrorResponse(responseDate, baseUrl, "badVerb", "Unknown verb: $verb"),
                ContentType.Text.Xml
            )
        }
    }
}

private fun isValidMetadataPrefix(prefix: String): Boolean {
    return prefix in listOf("oai_dc", "dcat", "epos_dcat_ap")
}

private fun buildIdentifyResponse(responseDate: String, baseUrl: String): String {
    return """<?xml version="1.0" encoding="UTF-8"?>
<OAI-PMH xmlns="http://www.openarchives.org/OAI/2.0/"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://www.openarchives.org/OAI/2.0/ http://www.openarchives.org/OAI/2.0/OAI-PMH.xsd">
  <responseDate>$responseDate</responseDate>
  <request verb="Identify">$baseUrl</request>
  <Identify>
    <repositoryName>EPOS Metadata Repository</repositoryName>
    <baseURL>$baseUrl</baseURL>
    <protocolVersion>2.0</protocolVersion>
    <adminEmail>support@epos-eu.org</adminEmail>
    <earliestDatestamp>2020-01-01T00:00:00Z</earliestDatestamp>
    <deletedRecord>no</deletedRecord>
    <granularity>YYYY-MM-DDThh:mm:ssZ</granularity>
    <description>
      <oai-identifier xmlns="http://www.openarchives.org/OAI/2.0/oai-identifier"
                      xsi:schemaLocation="http://www.openarchives.org/OAI/2.0/oai-identifier http://www.openarchives.org/OAI/2.0/oai-identifier.xsd">
        <scheme>oai</scheme>
        <repositoryIdentifier>epos-eu.org</repositoryIdentifier>
        <delimiter>:</delimiter>
        <sampleIdentifier>oai:epos-eu.org:dataset/12345</sampleIdentifier>
      </oai-identifier>
    </description>
  </Identify>
</OAI-PMH>"""
}

private fun buildListMetadataFormatsResponse(responseDate: String, baseUrl: String, identifier: String?): String {
    val requestAttrs = if (identifier != null) {
        """verb="ListMetadataFormats" identifier="$identifier""""
    } else {
        """verb="ListMetadataFormats""""
    }
    
    return """<?xml version="1.0" encoding="UTF-8"?>
<OAI-PMH xmlns="http://www.openarchives.org/OAI/2.0/"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://www.openarchives.org/OAI/2.0/ http://www.openarchives.org/OAI/2.0/OAI-PMH.xsd">
  <responseDate>$responseDate</responseDate>
  <request $requestAttrs>$baseUrl</request>
  <ListMetadataFormats>
    <metadataFormat>
      <metadataPrefix>oai_dc</metadataPrefix>
      <schema>http://www.openarchives.org/OAI/2.0/oai_dc.xsd</schema>
      <metadataNamespace>http://www.openarchives.org/OAI/2.0/oai_dc/</metadataNamespace>
    </metadataFormat>
    <metadataFormat>
      <metadataPrefix>dcat</metadataPrefix>
      <schema>http://www.w3.org/ns/dcat#</schema>
      <metadataNamespace>http://www.w3.org/ns/dcat#</metadataNamespace>
    </metadataFormat>
    <metadataFormat>
      <metadataPrefix>epos_dcat_ap</metadataPrefix>
      <schema>https://www.epos-eu.org/epos-dcat-ap</schema>
      <metadataNamespace>https://www.epos-eu.org/epos-dcat-ap#</metadataNamespace>
    </metadataFormat>
  </ListMetadataFormats>
</OAI-PMH>"""
}

private fun buildListSetsResponse(responseDate: String, baseUrl: String, shaclService: ShaclService): String {
    val resourceTypes = shaclService.getResourceTypes()
    
    val setsXml = resourceTypes.joinToString("\n    ") { type ->
        """<set>
      <setSpec>${type.name}</setSpec>
      <setName>${type.name.replaceFirstChar { it.uppercase() }}</setName>
    </set>"""
    }
    
    return """<?xml version="1.0" encoding="UTF-8"?>
<OAI-PMH xmlns="http://www.openarchives.org/OAI/2.0/"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://www.openarchives.org/OAI/2.0/ http://www.openarchives.org/OAI/2.0/OAI-PMH.xsd">
  <responseDate>$responseDate</responseDate>
  <request verb="ListSets">$baseUrl</request>
  <ListSets>
    $setsXml
  </ListSets>
</OAI-PMH>"""
}

private fun buildListIdentifiersResponse(
    responseDate: String,
    baseUrl: String,
    graphDBClient: GraphDBClient,
    metadataPrefix: String,
    set: String?,
    from: String?,
    until: String?,
    resumptionToken: String?
): String {
    // Parse resumption token for pagination
    val offset = resumptionToken?.toIntOrNull() ?: 0
    val limit = 100
    
    // Build SPARQL query
    val typeFilter = if (set != null) {
        "?s a ?type . FILTER(CONTAINS(LCASE(STR(?type)), \"${set.lowercase()}\"))"
    } else {
        "?s a ?type ."
    }
    
    val dateFilter = buildDateFilter(from, until)
    
    val query = """
        PREFIX dcterms: <http://purl.org/dc/terms/>
        SELECT DISTINCT ?s ?type ?modified
        FROM <${Config.GRAPH_PUBLISHED}>
        WHERE {
            $typeFilter
            OPTIONAL { ?s dcterms:modified ?modified }
            $dateFilter
        }
        ORDER BY ?s
        LIMIT $limit
        OFFSET $offset
    """.trimIndent()
    
    val results = graphDBClient.query(query)
    
    val headersXml = results.joinToString("\n    ") { row ->
        val identifier = "oai:epos-eu.org:${extractLocalId(row["s"] ?: "")}"
        val datestamp = row["modified"] ?: responseDate
        val setSpec = extractSetSpec(row["type"] ?: "")
        
        """<header>
      <identifier>$identifier</identifier>
      <datestamp>$datestamp</datestamp>
      <setSpec>$setSpec</setSpec>
    </header>"""
    }
    
    val resumption = if (results.size == limit) {
        """<resumptionToken>${offset + limit}</resumptionToken>"""
    } else {
        ""
    }
    
    return """<?xml version="1.0" encoding="UTF-8"?>
<OAI-PMH xmlns="http://www.openarchives.org/OAI/2.0/"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://www.openarchives.org/OAI/2.0/ http://www.openarchives.org/OAI/2.0/OAI-PMH.xsd">
  <responseDate>$responseDate</responseDate>
  <request verb="ListIdentifiers" metadataPrefix="$metadataPrefix"${set?.let { """ set="$it"""" } ?: ""}>$baseUrl</request>
  <ListIdentifiers>
    $headersXml
    $resumption
  </ListIdentifiers>
</OAI-PMH>"""
}

private fun buildListRecordsResponse(
    responseDate: String,
    baseUrl: String,
    graphDBClient: GraphDBClient,
    metadataPrefix: String,
    set: String?,
    from: String?,
    until: String?,
    resumptionToken: String?
): String {
    val offset = resumptionToken?.toIntOrNull() ?: 0
    val limit = 50
    
    val typeFilter = if (set != null) {
        "?s a ?type . FILTER(CONTAINS(LCASE(STR(?type)), \"${set.lowercase()}\"))"
    } else {
        "?s a ?type ."
    }
    
    val dateFilter = buildDateFilter(from, until)
    
    val query = """
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        
        SELECT DISTINCT ?s ?type ?title ?description ?modified ?creator ?subject
        FROM <${Config.GRAPH_PUBLISHED}>
        WHERE {
            $typeFilter
            OPTIONAL { ?s dcterms:title ?title }
            OPTIONAL { ?s dcterms:description ?description }
            OPTIONAL { ?s dcterms:modified ?modified }
            OPTIONAL { ?s dcterms:creator ?creator }
            OPTIONAL { ?s dcat:keyword ?subject }
            $dateFilter
        }
        ORDER BY ?s
        LIMIT $limit
        OFFSET $offset
    """.trimIndent()
    
    val results = graphDBClient.query(query)
    
    val recordsXml = results.joinToString("\n    ") { row ->
        buildRecordXml(row, metadataPrefix, responseDate)
    }
    
    val resumption = if (results.size == limit) {
        """<resumptionToken>${offset + limit}</resumptionToken>"""
    } else {
        ""
    }
    
    return """<?xml version="1.0" encoding="UTF-8"?>
<OAI-PMH xmlns="http://www.openarchives.org/OAI/2.0/"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://www.openarchives.org/OAI/2.0/ http://www.openarchives.org/OAI/2.0/OAI-PMH.xsd">
  <responseDate>$responseDate</responseDate>
  <request verb="ListRecords" metadataPrefix="$metadataPrefix"${set?.let { """ set="$it"""" } ?: ""}>$baseUrl</request>
  <ListRecords>
    $recordsXml
    $resumption
  </ListRecords>
</OAI-PMH>"""
}

private fun buildGetRecordResponse(
    responseDate: String,
    baseUrl: String,
    graphDBClient: GraphDBClient,
    identifier: String,
    metadataPrefix: String
): String {
    // Extract the actual URI from OAI identifier
    val entityUri = identifier.removePrefix("oai:epos-eu.org:")
    val fullUri = if (entityUri.startsWith("http")) entityUri else "${Config.BASE_URI}/$entityUri"
    
    val query = """
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        
        SELECT ?type ?title ?description ?modified ?creator ?subject
        FROM <${Config.GRAPH_PUBLISHED}>
        WHERE {
            <$fullUri> a ?type .
            OPTIONAL { <$fullUri> dcterms:title ?title }
            OPTIONAL { <$fullUri> dcterms:description ?description }
            OPTIONAL { <$fullUri> dcterms:modified ?modified }
            OPTIONAL { <$fullUri> dcterms:creator ?creator }
            OPTIONAL { <$fullUri> dcat:keyword ?subject }
        }
        LIMIT 1
    """.trimIndent()
    
    val results = graphDBClient.query(query)
    
    if (results.isEmpty()) {
        return buildErrorResponse(responseDate, baseUrl, "idDoesNotExist", "No record found for identifier: $identifier")
    }
    
    val row = results.first().toMutableMap()
    row["s"] = fullUri
    
    val recordXml = buildRecordXml(row, metadataPrefix, responseDate)
    
    return """<?xml version="1.0" encoding="UTF-8"?>
<OAI-PMH xmlns="http://www.openarchives.org/OAI/2.0/"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://www.openarchives.org/OAI/2.0/ http://www.openarchives.org/OAI/2.0/OAI-PMH.xsd">
  <responseDate>$responseDate</responseDate>
  <request verb="GetRecord" identifier="$identifier" metadataPrefix="$metadataPrefix">$baseUrl</request>
  <GetRecord>
    $recordXml
  </GetRecord>
</OAI-PMH>"""
}

private fun buildRecordXml(row: Map<String, String>, metadataPrefix: String, responseDate: String): String {
    val identifier = "oai:epos-eu.org:${extractLocalId(row["s"] ?: "")}"
    val datestamp = row["modified"] ?: responseDate
    val setSpec = extractSetSpec(row["type"] ?: "")
    
    val metadata = when (metadataPrefix) {
        "oai_dc" -> buildDublinCoreMetadata(row)
        "dcat" -> buildDcatMetadata(row)
        "epos_dcat_ap" -> buildEposDcatApMetadata(row)
        else -> buildDublinCoreMetadata(row)
    }
    
    return """<record>
      <header>
        <identifier>$identifier</identifier>
        <datestamp>$datestamp</datestamp>
        <setSpec>$setSpec</setSpec>
      </header>
      <metadata>
        $metadata
      </metadata>
    </record>"""
}

private fun buildDublinCoreMetadata(row: Map<String, String>): String {
    val title = escapeXml(row["title"] ?: "")
    val description = escapeXml(row["description"] ?: "")
    val creator = escapeXml(row["creator"] ?: "")
    val subject = escapeXml(row["subject"] ?: "")
    val identifier = escapeXml(row["s"] ?: "")
    val type = extractSetSpec(row["type"] ?: "")
    
    return """<oai_dc:dc xmlns:oai_dc="http://www.openarchives.org/OAI/2.0/oai_dc/"
                  xmlns:dc="http://purl.org/dc/elements/1.1/"
                  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
                  xsi:schemaLocation="http://www.openarchives.org/OAI/2.0/oai_dc/ http://www.openarchives.org/OAI/2.0/oai_dc.xsd">
          ${if (title.isNotEmpty()) "<dc:title>$title</dc:title>" else ""}
          ${if (creator.isNotEmpty()) "<dc:creator>$creator</dc:creator>" else ""}
          ${if (subject.isNotEmpty()) "<dc:subject>$subject</dc:subject>" else ""}
          ${if (description.isNotEmpty()) "<dc:description>$description</dc:description>" else ""}
          <dc:identifier>$identifier</dc:identifier>
          <dc:type>$type</dc:type>
        </oai_dc:dc>"""
}

private fun buildDcatMetadata(row: Map<String, String>): String {
    val uri = row["s"] ?: ""
    val title = escapeXml(row["title"] ?: "")
    val description = escapeXml(row["description"] ?: "")
    
    return """<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
                xmlns:dcat="http://www.w3.org/ns/dcat#"
                xmlns:dcterms="http://purl.org/dc/terms/">
          <dcat:Resource rdf:about="$uri">
            ${if (title.isNotEmpty()) "<dcterms:title>$title</dcterms:title>" else ""}
            ${if (description.isNotEmpty()) "<dcterms:description>$description</dcterms:description>" else ""}
          </dcat:Resource>
        </rdf:RDF>"""
}

private fun buildEposDcatApMetadata(row: Map<String, String>): String {
    val uri = row["s"] ?: ""
    val type = row["type"] ?: ""
    val title = escapeXml(row["title"] ?: "")
    val description = escapeXml(row["description"] ?: "")
    
    return """<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
                xmlns:dcat="http://www.w3.org/ns/dcat#"
                xmlns:dcterms="http://purl.org/dc/terms/"
                xmlns:epos="https://www.epos-eu.org/epos-dcat-ap#">
          <rdf:Description rdf:about="$uri">
            <rdf:type rdf:resource="$type"/>
            ${if (title.isNotEmpty()) "<dcterms:title>$title</dcterms:title>" else ""}
            ${if (description.isNotEmpty()) "<dcterms:description>$description</dcterms:description>" else ""}
          </rdf:Description>
        </rdf:RDF>"""
}

private fun buildErrorResponse(responseDate: String, baseUrl: String, code: String, message: String): String {
    return """<?xml version="1.0" encoding="UTF-8"?>
<OAI-PMH xmlns="http://www.openarchives.org/OAI/2.0/"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://www.openarchives.org/OAI/2.0/ http://www.openarchives.org/OAI/2.0/OAI-PMH.xsd">
  <responseDate>$responseDate</responseDate>
  <request>$baseUrl</request>
  <error code="$code">$message</error>
</OAI-PMH>"""
}

private fun buildDateFilter(from: String?, until: String?): String {
    val filters = mutableListOf<String>()
    if (from != null) {
        filters.add("FILTER(!BOUND(?modified) || ?modified >= \"$from\"^^xsd:dateTime)")
    }
    if (until != null) {
        filters.add("FILTER(!BOUND(?modified) || ?modified <= \"$until\"^^xsd:dateTime)")
    }
    return filters.joinToString(" ")
}

private fun extractLocalId(uri: String): String {
    return uri.substringAfterLast("/").substringAfterLast("#")
}

private fun extractSetSpec(typeUri: String): String {
    return typeUri.substringAfterLast("/").substringAfterLast("#").lowercase()
}

private fun escapeXml(text: String): String {
    return text
        .replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace("\"", "&quot;")
        .replace("'", "&apos;")
}
