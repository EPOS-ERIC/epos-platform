package common

import "testing"

func TestIssueTrackerValidate(t *testing.T) {
	const absolute = "https://example.com/issues"
	for _, tt := range []struct {
		name    string
		tracker IssueTracker
		wantErr bool
	}{
		{name: "disabled"},
		{name: "configured", tracker: IssueTracker{URL: absolute, OpenNew: absolute + "/new"}},
		{name: "missing URL", tracker: IssueTracker{OpenNew: absolute}, wantErr: true},
		{name: "missing create URL", tracker: IssueTracker{URL: absolute}, wantErr: true},
		{name: "relative URL", tracker: IssueTracker{URL: "/issues", OpenNew: absolute}, wantErr: true},
		{name: "invalid create URL", tracker: IssueTracker{URL: absolute, OpenNew: "%zz"}, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.tracker.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, want error: %t", err, tt.wantErr)
			}
		})
	}
}
