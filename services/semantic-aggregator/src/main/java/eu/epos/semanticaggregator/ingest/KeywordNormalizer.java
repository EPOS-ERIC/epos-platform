package eu.epos.semanticaggregator.ingest;

import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;

/** Splits legacy comma-delimited keyword literals into individual keywords. */
final class KeywordNormalizer {
    private KeywordNormalizer() {}

    static Result normalize(List<String> keywords) {
        List<String> result = new ArrayList<>();
        var warnings = new LinkedHashSet<String>();
        for (String keyword : keywords) {
            if (keyword.contains(",")) warnings.add("split comma-delimited keyword");
            for (String value : keyword.split(",", -1)) {
                String trimmed = value.trim();
                if (!value.equals(trimmed)) warnings.add("trimmed keyword whitespace");
                if (trimmed.isEmpty()) {
                    warnings.add("ignored empty keyword");
                } else {
                    result.add(trimmed);
                }
            }
        }
        return new Result(List.copyOf(result), List.copyOf(warnings));
    }

    record Result(List<String> values, List<String> warnings) {}
}
