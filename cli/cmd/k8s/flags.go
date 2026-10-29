package k8s

import "time"

var (
	configFilePath   string
	context          string
	timeout          time.Duration
	populateExamples bool
	deleteForce      bool
)
