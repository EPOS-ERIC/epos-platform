package eu.epos.semanticaggregator.aggregation;

import java.io.IOException;
import org.apache.jena.rdf.model.Model;

/** Replaces one named graph; ownership of the model stays with the caller. */
@FunctionalInterface
public interface GraphPublisher {
    void replace(String graphIri, Model model) throws IOException, InterruptedException;
}
