package eu.epos.semanticaggregator.upstream;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import eu.epos.semanticaggregator.ingest.DatasetMapper;
import eu.epos.semanticaggregator.rdf.TurtleReader;
import java.io.ByteArrayInputStream;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import metadata_lifecycle.v1.Resource;
import org.junit.jupiter.api.Test;

class DatasetRequestMapperTest {
    private static final String PREFIXES = """
            @prefix dcat: <http://www.w3.org/ns/dcat#> .
            @prefix dct: <http://purl.org/dc/terms/> .
            @prefix schema: <http://schema.org/> .
            @prefix adms: <http://www.w3.org/ns/adms#> .
            @prefix skos: <http://www.w3.org/2004/02/skos/core#> .
            @prefix ex: <https://example.org/> .
            """;

    @Test
    void convertsNestedRdfValuesAndOpenTemporalBoundToCreateResource() {
        var model = read(PREFIXES + """
                ex:dataset a dcat:Dataset; dct:title "Title"; dct:description "Description";
                    dcat:keyword "earthquake";
                    dct:temporal [ schema:startDate "2020-02-03T12:30:00+01:00" ];
                    adms:identifier [ adms:schemeAgency "DOI"; skos:notation "10.123/example" ] .
                """);
        try {
            var mapping = mapper().map(model).getFirst();
            var request = new DatasetRequestMapper().toRequest(mapping, "123e4567-e89b-12d3-a456-426614174000");
            assertEquals("Title", request.getDataset().getTitle());
            assertEquals("DOI", request.getDataset().getPersistentIdentifiers(0).getType());
            assertTrue(request.getDataset().getTemporalCoverage().hasStart());
            assertFalse(request.getDataset().getTemporalCoverage().hasEnd());
            assertEquals(Instant.parse("2020-02-03T11:30:00Z").getEpochSecond(),
                    request.getDataset().getTemporalCoverage().getStart().getSeconds());
            assertThrows(IllegalArgumentException.class, () -> new DatasetRequestMapper().toRequest(mapping, "1-1-1-1-1"));
        } finally {
            model.close();
        }
    }

    @Test
    void preservesPrimaryIdentifierValueAndDatatypeInConfiguredTypes() {
        var model = read(PREFIXES + """
                @prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
                ex:a a dcat:Dataset; dct:title "A"; dct:description "A";
                    dct:identifier "urn:example:first"^^xsd:anyURI .
                ex:b a dcat:Dataset; dct:title "B"; dct:description "B";
                    dct:identifier "dataset-b" .
                """);
        try {
            var datasets = mapper().map(model);
            var converter = new DatasetRequestMapper();
            var first = converter.toRequest(datasets.get(0), "123e4567-e89b-12d3-a456-426614174000").getDataset();
            var second = converter.toRequest(datasets.get(1), "123e4567-e89b-12d3-a456-426614174000").getDataset();
            assertEquals("EPOS-DCAT-AP:dct:identifier:xsd:anyURI", first.getPersistentIdentifiers(0).getType());
            assertEquals("urn:example:first", first.getPersistentIdentifiers(0).getValue());
            assertEquals("EPOS-DCAT-AP:dct:identifier:xsd:string", second.getPersistentIdentifiers(0).getType());
            assertEquals("dataset-b", second.getPersistentIdentifiers(0).getValue());
        } finally {
            model.close();
        }
    }

    @Test
    void rejectsMultipleOrUnsupportedPrimaryIdentifiers() {
        var multiple = read(PREFIXES + "ex:a a dcat:Dataset; dct:title \"A\"; dct:description \"A\"; dct:identifier \"one\", \"two\" .");
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(multiple));
        } finally {
            multiple.close();
        }
        var unsupported = read(PREFIXES + "ex:a a dcat:Dataset; dct:title \"A\"; dct:description \"A\"; dct:identifier 42 .");
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(unsupported));
        } finally {
            unsupported.close();
        }
    }

    @Test
    void ignoresUnsupportedDatasetProperties() {
        var model = read(PREFIXES + "ex:dataset a dcat:Dataset; dct:title \"Title\"; "
                + "dct:description \"Description\"; ex:unknown \"value\" .");
        try {
            var mapping = mapper().map(model).getFirst();
            assertEquals(Resource.CreateResourceRequest.MetadataCase.DATASET,
                    new DatasetRequestMapper().toRequest(mapping, "123e4567-e89b-12d3-a456-426614174000").getMetadataCase());
        } finally {
            model.close();
        }
    }

    @Test
    void mapsDownloadableDistributionAndDatasetMetadata() {
        var model = read(PREFIXES + """
                ex:dataset a dcat:Dataset; dct:title "Title"; dct:description "Description";
                    dct:created "2020-01-01"; dct:accrualPeriodicity <https://example.org/daily>;
                    <http://www.w3.org/ns/dqv#hasQualityAnnotation>
                        [ <http://www.w3.org/ns/oa#hasBody> <https://example.org/quality> ];
                    dcat:distribution [ a dcat:Distribution; dct:title "CSV"; dct:description "Download";
                        dct:type <http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE>;
                        dcat:downloadURL <https://example.org/data.csv>; dct:format "text/csv";
                        dct:license <https://example.org/license>; dcat:byteSize 42 ] .
                """);
        try {
            var mapping = mapper().map(model).getFirst();
            var dataset = new DatasetRequestMapper().toRequest(mapping, "123e4567-e89b-12d3-a456-426614174000").getDataset();
            assertEquals("https://example.org/daily", dataset.getAccrualPeriodicity());
            assertEquals("https://example.org/quality", dataset.getQualityAnnotation());
            assertEquals(Instant.parse("2020-01-01T00:00:00Z").getEpochSecond(), dataset.getCreated().getSeconds());
            assertEquals("DOWNLOADABLE_FILE", dataset.getDistributions(0).getType());
            assertEquals("https://example.org/data.csv", dataset.getDistributions(0).getDownloadableFile().getUrl());
            assertEquals("text/csv", dataset.getDistributions(0).getDownloadableFile().getFormat());
            assertEquals(42, dataset.getDistributions(0).getByteSize());
        } finally {
            model.close();
        }
    }

    @Test
    void acceptsAccessUrlForDownloadableFilesAndRejectsConflictingUrls() {
        String prefix = PREFIXES + """
                ex:dataset a dcat:Dataset; dct:title "Title"; dct:description "Description";
                    dcat:distribution [ dct:title "CSV"; dct:description "Download";
                        dct:type "DOWNLOADABLE_FILE"; dcat:accessURL <https://example.org/a.csv>;
                        dct:format "text/csv"%s ] .
                """;
        var model = read(prefix.formatted(""));
        try {
            var mapping = mapper().map(model).getFirst();
            var request = new DatasetRequestMapper().toRequest(mapping, "123e4567-e89b-12d3-a456-426614174000");
            assertEquals("https://example.org/a.csv", request.getDataset().getDistributions(0).getDownloadableFile().getUrl());
        } finally {
            model.close();
        }
        var conflicting = read(prefix.formatted("; dcat:downloadURL <https://example.org/b.csv>"));
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(conflicting));
        } finally {
            conflicting.close();
        }
    }

    @Test
    void rejectsWebServiceRatherThanTreatingItAsDownload() {
        var model = read(PREFIXES + """
                ex:dataset a dcat:Dataset; dct:title "Title"; dct:description "Description";
                    dcat:distribution [ dct:type "WEB_SERVICE" ] .
                """);
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(model));
        } finally {
            model.close();
        }
    }

    @Test
    void rejectsReversedTemporalRange() {
        var model = read(PREFIXES + """
                ex:dataset a dcat:Dataset; dct:title "Title"; dct:description "Description";
                    dct:temporal [ schema:startDate "2021-01-01"; schema:endDate "2020-01-01" ] .
                """);
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(model));
        } finally {
            model.close();
        }
    }

    private static org.apache.jena.rdf.model.Model read(String turtle) {
        return new TurtleReader().read(new ByteArrayInputStream(turtle.getBytes(StandardCharsets.UTF_8)),
                "https://example.org/");
    }

    private static DatasetMapper mapper() {
        try {
            return new DatasetMapper(eu.epos.semanticaggregator.mapping.MappingRegistry.withDefaults()
                    .get("epos-dcat-common", 1));
        } catch (java.io.IOException exception) {
            throw new java.io.UncheckedIOException(exception);
        }
    }
}
