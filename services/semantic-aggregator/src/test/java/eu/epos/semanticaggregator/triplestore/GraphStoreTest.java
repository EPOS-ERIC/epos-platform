package eu.epos.semanticaggregator.triplestore;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.net.URI;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.atomic.AtomicReference;
import org.apache.jena.rdf.model.ModelFactory;
import org.junit.jupiter.api.Test;

class GraphStoreTest {
    @Test
    void sendsNamedGraphAndDoesNotExposeErrorBody() throws Exception {
        var query = new AtomicReference<String>();
        var authorization = new AtomicReference<String>();
        var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/ds/data", exchange -> {
            try (exchange) {
                query.set(exchange.getRequestURI().getRawQuery());
                authorization.set(exchange.getRequestHeaders().getFirst("Authorization"));
                byte[] body = "sensitive upstream response".getBytes(StandardCharsets.UTF_8);
                exchange.sendResponseHeaders(403, body.length);
                exchange.getResponseBody().write(body);
            }
        });
        server.start();
        var store = new GraphStore(URI.create("http://127.0.0.1:" + server.getAddress().getPort() + "/ds/data"), "test-token");
        var model = ModelFactory.createDefaultModel();
        try {
            var error = assertThrows(IOException.class, () -> store.replace("urn:example:graph?x=1", model));
            assertTrue(error.getMessage().contains("403"), error.getMessage());
            assertFalse(error.getMessage().contains("sensitive"));
            assertEquals("graph=urn%3Aexample%3Agraph%3Fx%3D1", query.get());
            assertEquals("Bearer test-token", authorization.get());
        } finally {
            model.close();
            server.stop(0);
        }
    }

    @Test
    void rejectsInvalidRepositoryAndGraphUris() {
        assertThrows(IllegalArgumentException.class, () -> new GraphStore(URI.create("http://user:pass@example.org/repositories/x"), null));
        var store = new GraphStore(URI.create("http://localhost/ds/data"), null);
        var model = ModelFactory.createDefaultModel();
        try {
            assertThrows(IllegalArgumentException.class, () -> store.replace("relative", model));
        } finally {
            model.close();
        }
    }

    @Test
    void basicAndBearerAuthenticationAreExplicit() throws Exception {
        var authorization = new AtomicReference<String>();
        var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/data", exchange -> {
            try (exchange) {
                authorization.set(exchange.getRequestHeaders().getFirst("Authorization"));
                exchange.sendResponseHeaders(204, -1);
            }
        });
        server.start();
        var endpoint = URI.create("http://127.0.0.1:" + server.getAddress().getPort() + "/data");
        var model = ModelFactory.createDefaultModel();
        try {
            new GraphStore(endpoint, "token:with-colon").replace("urn:example:graph", model);
            assertEquals("Bearer token:with-colon", authorization.get());
            GraphStore.withBasicAuth(endpoint, "test-user", "test-password").replace("urn:example:graph", model);
            assertEquals("Basic " + java.util.Base64.getEncoder().encodeToString(
                    "test-user:test-password".getBytes(StandardCharsets.UTF_8)), authorization.get());
        } finally {
            model.close();
            server.stop(0);
        }
    }
}
