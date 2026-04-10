package org.epos.infrastructure.ktor.plugins

object OpenApiOaiPmhPaths {
    val yaml = """
  # ============================================================
  # OAI-PMH ENDPOINT
  # ============================================================
  /oai-pmh:
    get:
      tags: [OAI-PMH]
      summary: OAI-PMH 2.0 endpoint
      description: |
        OAI-PMH 2.0 compliant endpoint for metadata harvesting.

        Supported verbs: Identify, ListMetadataFormats, ListSets, ListIdentifiers, ListRecords, GetRecord

        Supported metadata formats: oai_dc (Dublin Core), dcat (DCAT-AP), epos_dcat_ap (EPOS-DCAT-AP)
      parameters:
        - name: verb
          in: query
          required: true
          description: OAI-PMH verb
          schema:
            type: string
            enum: [Identify, ListMetadataFormats, ListSets, ListIdentifiers, ListRecords, GetRecord]
        - name: metadataPrefix
          in: query
          description: Metadata format (required for ListIdentifiers, ListRecords, GetRecord)
          schema:
            type: string
            enum: [oai_dc, dcat, epos_dcat_ap]
        - name: identifier
          in: query
          description: Record identifier (required for GetRecord)
          schema:
            type: string
        - name: set
          in: query
          description: Set specification for selective harvesting
          schema:
            type: string
        - name: from
          in: query
          description: Start date for selective harvesting (ISO 8601)
          schema:
            type: string
            format: date-time
        - name: until
          in: query
          description: End date for selective harvesting (ISO 8601)
          schema:
            type: string
            format: date-time
        - name: resumptionToken
          in: query
          description: Token for paginated results
          schema:
            type: string
      responses:
        '200':
          description: OAI-PMH XML response
          content:
            text/xml:
              schema:
                type: string"""
}
