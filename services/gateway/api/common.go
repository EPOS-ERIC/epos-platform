package api

type ErrorResponse struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

type ContactPoint struct {
	ID                string
	Email             string
	AvailableLanguage string
	ContactType       string
}
