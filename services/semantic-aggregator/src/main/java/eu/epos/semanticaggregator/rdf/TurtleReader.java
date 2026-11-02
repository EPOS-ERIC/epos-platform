package eu.epos.semanticaggregator.rdf;

import java.io.InputStream;
import java.util.Objects;
import org.apache.jena.rdf.model.Model;
import org.apache.jena.rdf.model.ModelFactory;
import org.apache.jena.riot.Lang;
import org.apache.jena.riot.RDFParser;

/** Loads one Turtle document into an isolated in-memory model. */
public final class TurtleReader {
    /** The caller owns and must close the returned model and the input stream. */
    public Model read(InputStream input, String baseUri) {
        Objects.requireNonNull(input, "input");
        Objects.requireNonNull(baseUri, "baseUri");
        Model model = ModelFactory.createDefaultModel();
        try {
            RDFParser.create().source(input).base(baseUri).lang(Lang.TURTLE).parse(model);
            return model;
        } catch (RuntimeException exception) {
            model.close();
            throw exception;
        }
    }
}
