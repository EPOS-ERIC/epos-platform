package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ogen-go/ogen/ogenerrors"
)

// decodedURITarget recovers array indices omitted by ogen's JSON decoder.
// Ambiguous matches are omitted rather than reporting the wrong field.
// (this is a hack but it kind of works)
func decodedURITarget(err error) string {
	bodyErr, ok := errors.AsType[*ogenerrors.DecodeBodyError](err)
	if !ok {
		return ""
	}
	uriErr, ok := errors.AsType[*url.Error](bodyErr.Err)
	if !ok {
		return ""
	}

	var fields []string
	for current := bodyErr.Err; current != nil; current = errors.Unwrap(current) {
		next := errors.Unwrap(current)
		if next == nil {
			break
		}
		prefix, ok := strings.CutSuffix(current.Error(), ": "+next.Error())
		if !ok {
			continue
		}
		quoted, ok := strings.CutPrefix(prefix, "decode field ")
		if !ok {
			continue
		}
		field, err := strconv.Unquote(quoted)
		if err != nil {
			return ""
		}
		fields = append(fields, field)
	}
	if len(fields) == 0 {
		return ""
	}

	var body any
	if json.Unmarshal(bodyErr.Body, &body) != nil {
		return ""
	}
	matches := uriTargets(body, fields, uriErr.URL, "")
	if len(matches) != 1 {
		return ""
	}
	return matches[0]
}

func uriTargets(value any, fields []string, invalid, path string) []string {
	if items, ok := value.([]any); ok {
		var matches []string
		for i, item := range items {
			matches = append(matches, uriTargets(item, fields, invalid, fmt.Sprintf("%s[%d]", path, i))...)
		}
		return matches
	}
	if len(fields) == 0 {
		if text, ok := value.(string); ok && text == invalid {
			return []string{path}
		}
		return nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	next, ok := object[fields[0]]
	if !ok {
		return nil
	}
	if path != "" {
		path += "."
	}
	return uriTargets(next, fields[1:], invalid, path+fields[0])
}
