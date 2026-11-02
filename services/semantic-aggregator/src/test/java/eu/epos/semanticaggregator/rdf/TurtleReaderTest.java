package eu.epos.semanticaggregator.rdf;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.io.ByteArrayInputStream;
import java.nio.charset.StandardCharsets;
import org.apache.jena.riot.RiotException;
import org.junit.jupiter.api.Test;

class TurtleReaderTest {
    @Test
    void readsLinkedBlankNodesIntoAnIsolatedGraph() {
        String turtle = "@prefix ex: <https://example.org/> . ex:s ex:p [ ex:q \"value\" ] .";
        var input = new ByteArrayInputStream(turtle.getBytes(StandardCharsets.UTF_8));
        var model = new TurtleReader().read(input, "https://example.org/");
        try {
            assertEquals(2, model.size());
        } finally {
            model.close();
        }
    }

    @Test
    void rejectsInvalidTurtle() {
        var input = new ByteArrayInputStream("not turtle".getBytes(StandardCharsets.UTF_8));
        assertThrows(RiotException.class, () -> new TurtleReader().read(input, "https://example.org/"));
    }
}
