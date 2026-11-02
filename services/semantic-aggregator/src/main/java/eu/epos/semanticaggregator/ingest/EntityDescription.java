package eu.epos.semanticaggregator.ingest;

import java.util.Map;

/** RDF attributes that must match an existing canonical entity. */
public record EntityDescription(DatasetMapping.Reference reference, Map<String, String> fields) {
    public EntityDescription {
        fields = Map.copyOf(fields);
    }
}
