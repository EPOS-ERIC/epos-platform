package common

import "strings"

type URLs struct {
	GUIURL        string
	APIDocsURL    string
	APIBaseURL    string
	BackofficeURL *string
	GrafanaURL    *string
}

func (u *URLs) String() string {
	if u == nil {
		return ""
	}

	parts := []string{
		"gui=" + u.GUIURL,
		"api=" + u.APIBaseURL,
		"api_docs=" + u.APIDocsURL,
	}
	if u.BackofficeURL != nil && *u.BackofficeURL != "" {
		parts = append(parts, "backoffice="+*u.BackofficeURL)
	}

	if u.GrafanaURL != nil && *u.GrafanaURL != "" {
		parts = append(parts, "grafana="+*u.GrafanaURL)
	}

	return strings.Join(parts, ", ")
}
