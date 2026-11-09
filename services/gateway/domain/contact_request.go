package domain

// ContactRequestEmail contains the fields Gateway needs to deliver a prepared email.
type ContactRequestEmail struct {
	TargetEmail string
	Subject     string
	Body        string
}
