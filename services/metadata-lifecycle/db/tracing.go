package db

import "strings"

func queryOperationName(statement string) string {
	for {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			return "UNKNOWN"
		}

		line, rest, hasNextLine := strings.Cut(statement, "\n")
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "--" && fields[1] == "name:" {
			return fields[2]
		}
		if !strings.HasPrefix(line, "--") || !hasNextLine {
			break
		}
		statement = rest
	}

	for word := range strings.FieldsSeq(statement) {
		return strings.ToUpper(word)
	}
	return "UNKNOWN"
}
