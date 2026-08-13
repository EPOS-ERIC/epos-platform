// Package authorization provides PostgreSQL persistence for authorization decisions.
package authorization

import "github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"

type Repository struct {
	*sqlc.Queries
}
