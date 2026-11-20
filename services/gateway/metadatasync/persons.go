package metadatasync

import (
	"context"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type remotePerson struct {
	NID   []remoteInt    `json:"nid"`
	Email []remoteString `json:"field_email"`
}

func (w *Worker) fetchContactPoints(ctx context.Context) ([]domain.SyncedContactPoint, int, error) {
	rows, pages, err := fetchSnapshot[remotePerson](ctx, w, "api/persons")
	if err != nil {
		return nil, pages, err
	}

	values := make([]domain.SyncedContactPoint, 0, len(rows))
	seen := make(map[int64]bool, len(rows))
	for i, person := range rows {
		if len(person.NID) != 1 || person.NID[0].Value <= 0 {
			return values, pages, fmt.Errorf("invalid person NID at row %d", i)
		}

		nid := person.NID[0].Value
		if seen[nid] {
			return values, pages, fmt.Errorf("duplicate person NID %d at row %d", nid, i)
		}

		seen[nid] = true

		if len(person.Email) > 1 {
			return values, pages, fmt.Errorf("person %d has multiple primary emails", nid)
		}

		var email string
		if len(person.Email) == 1 {
			email = strings.TrimSpace(person.Email[0].Value)
		}
		if email == "" {
			w.log.DebugContext(ctx, "skip Drupal person without primary email", "nid", nid)
			continue
		}

		values = append(values, domain.SyncedContactPoint{NID: nid, Email: email})
	}

	return values, pages, nil
}
