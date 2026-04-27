package org.epos.graphdb.services

import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.epos.graphdb.utils.*

/**
 * Service for recording PROV-O provenance information.
 * Optimized version using shared RDF utilities.
 */
class ProvenanceService(private val graphDBClient: GraphDBClient) {

    /**
     * Records the initial creation of an entity from an external source.
     */
    fun recordInitialCreation(
        versionUri: String,
        conceptualUri: String,
        source: String,
        agent: String = Config.AGENT_INITIAL_POPULATION,
        group: String = Config.AGENT_SYSTEM
    ) {
        val activityUri = UriUtils.generateActivityUri()
        val timestamp = now()
        val safeAgent = UriUtils.ensureAbsolute(agent, "agent")
        val safeGroup = UriUtils.ensureAbsolute(group, "group")
        val safeSource = UriUtils.ensureAbsolute(source, "source")

        val provModel = model {
            // Version entity metadata
            subject(versionUri) {
                generatedBy(activityUri)
                generatedAt(timestamp)
                attributedTo(safeAgent)
                isVersionOf(conceptualUri)
            }
            // Activity metadata
            subject(activityUri) {
                type(NS.PROV_ACTIVITY)
                label("Initial import from source")
                used(safeSource)
                associatedWith(safeAgent)
                associatedWith(safeGroup)
                startedAt(timestamp)
            }
        }

        graphDBClient.insertIntoGraph(provModel, Config.GRAPH_PROVENANCE)
    }

    /**
     * Records the creation of a new draft version based on an existing version.
     */
    fun recordDraftCreation(
        newVersionUri: String,
        basedOnVersionUri: String,
        conceptualUri: String,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM
    ) {
        val activityUri = UriUtils.generateActivityUri()
        val timestamp = now()
        val safeAgent = UriUtils.ensureAbsolute(agent, "agent")
        val safeGroup = UriUtils.ensureAbsolute(group, "group")

        val provModel = model {
            // New version metadata
            subject(newVersionUri) {
                revisionOf(basedOnVersionUri)
                generatedBy(activityUri)
                generatedAt(timestamp)
                attributedTo(safeAgent)
                isVersionOf(conceptualUri)
            }
            // Activity metadata
            subject(activityUri) {
                type(NS.PROV_ACTIVITY)
                label("Draft creation")
                used(basedOnVersionUri)
                associatedWith(safeAgent)
                associatedWith(safeGroup)
                startedAt(timestamp)
            }
        }

        graphDBClient.insertIntoGraph(provModel, Config.GRAPH_PROVENANCE)
    }

    /**
     * Records a state transition (e.g., DRAFT -> SUBMITTED -> PUBLISHED).
     */
    fun recordStateTransition(
        versionUri: String,
        fromStatus: String,
        toStatus: String,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM
    ) {
        val activityUri = UriUtils.generateActivityUri()
        val timestamp = now()
        val safeAgent = UriUtils.ensureAbsolute(agent, "agent")
        val safeGroup = UriUtils.ensureAbsolute(group, "group")

        val provModel = model {
            // Activity metadata
            subject(activityUri) {
                type(NS.PROV_ACTIVITY)
                label("State transition: $fromStatus -> $toStatus")
                used(versionUri)
                associatedWith(safeAgent)
                associatedWith(safeGroup)
                startedAt(timestamp)
            }
            // Link version to activity
            subject(versionUri) {
                influencedBy(activityUri)
            }
        }

        graphDBClient.insertIntoGraph(provModel, Config.GRAPH_PROVENANCE)
    }

    /**
     * Retrieves the complete version history of a conceptual entity.
     */
    fun getVersionHistory(conceptualUri: String): List<Map<String, String>> =
        graphDBClient.query(Sparql.selectVersionHistory(conceptualUri))
}
