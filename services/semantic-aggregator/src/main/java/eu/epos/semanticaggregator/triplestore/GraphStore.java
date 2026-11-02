package eu.epos.semanticaggregator.triplestore;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.Objects;
import java.util.Optional;
import org.apache.jena.rdf.model.Model;
import org.apache.jena.rdf.model.ModelFactory;
import org.apache.jena.riot.Lang;
import org.apache.jena.riot.RDFDataMgr;
import eu.epos.semanticaggregator.aggregation.GraphPublisher;

/** SPARQL Graph Store Protocol adapter for named graphs. */
public final class GraphStore implements GraphPublisher {
    private final URI repository;
    private final HttpClient client;
    private final String authorization;

    public GraphStore(URI repository, String bearerToken) {
        this(repository, bearerToken == null || bearerToken.isBlank() ? null : "Bearer " + bearerToken,
                HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(10)).build());
    }

    public static GraphStore withBasicAuth(URI repository, String username, String password) {
        if (username == null || username.isBlank() || username.contains(":") || password == null) {
            throw new IllegalArgumentException("invalid basic authentication credentials");
        }
        String credentials = java.util.Base64.getEncoder().encodeToString((username + ":" + password).getBytes(StandardCharsets.UTF_8));
        return new GraphStore(repository, "Basic " + credentials,
                HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(10)).build());
    }

    private GraphStore(URI repository, String authorization, HttpClient client) {
        Objects.requireNonNull(repository, "repository");
        if (!("http".equalsIgnoreCase(repository.getScheme()) || "https".equalsIgnoreCase(repository.getScheme()))
                || repository.getHost() == null || repository.getRawQuery() != null || repository.getRawFragment() != null
                || repository.getRawUserInfo() != null) {
            throw new IllegalArgumentException("repository must be an HTTP(S) URL without credentials, query or fragment");
        }
        this.repository = repository;
        this.client = Objects.requireNonNull(client, "client");
        this.authorization = authorization;
    }

    /** Replaces one graph with a single Graph Store PUT. */
    @Override
    public void replace(String graphIri, Model model) throws IOException, InterruptedException {
        Objects.requireNonNull(model, "model");
        var output = new ByteArrayOutputStream();
        RDFDataMgr.write(output, model, Lang.NTRIPLES);
        var request = request(graphIri).header("Content-Type", "application/n-triples")
                .PUT(HttpRequest.BodyPublishers.ofByteArray(output.toByteArray())).build();
        requireSuccess(client.send(request, HttpResponse.BodyHandlers.discarding()).statusCode());
    }

    /** Returns an independently owned model, or empty if the named graph does not exist. */
    public Optional<Model> read(String graphIri) throws IOException, InterruptedException {
        var request = request(graphIri).header("Accept", "application/n-triples").GET().build();
        var response = client.send(request, HttpResponse.BodyHandlers.ofInputStream());
        try (var body = response.body()) {
            if (response.statusCode() == 404) return Optional.empty();
            requireSuccess(response.statusCode());
            Model model = ModelFactory.createDefaultModel();
            try {
                RDFDataMgr.read(model, body, Lang.NTRIPLES);
                return Optional.of(model);
            } catch (RuntimeException exception) {
                model.close();
                throw exception;
            }
        }
    }

    /** Removes only the named graph. */
    public void delete(String graphIri) throws IOException, InterruptedException {
        var request = request(graphIri).DELETE().build();
        int status = client.send(request, HttpResponse.BodyHandlers.discarding()).statusCode();
        if (status != 404) requireSuccess(status);
    }

    private HttpRequest.Builder request(String graphIri) {
        URI uri = URI.create(graphIri);
        if (!uri.isAbsolute()) throw new IllegalArgumentException("named graph IRI must be absolute");
        String base = repository.toString();
        if (base.endsWith("/")) base = base.substring(0, base.length() - 1);
        URI endpoint = URI.create(base + "?graph=" + URLEncoder.encode(graphIri, StandardCharsets.UTF_8));
        var builder = HttpRequest.newBuilder(endpoint).timeout(Duration.ofMinutes(2));
        if (authorization != null) builder.header("Authorization", authorization);
        return builder;
    }

    private static void requireSuccess(int status) throws IOException {
        if (status < 200 || status >= 300) throw new IOException("Graph Store request failed with HTTP status " + status);
    }
}
