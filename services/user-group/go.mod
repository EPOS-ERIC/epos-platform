module github.com/EPOS-ERIC/epos-platform/services/user-group

go 1.26.2

require github.com/jackc/pgx/v5 v5.7.6

require github.com/EPOS-ERIC/epos-platform/packages/go/logging v0.0.0
require github.com/EPOS-ERIC/epos-platform/packages/go/migrations v0.0.0
require github.com/EPOS-ERIC/epos-platform/packages/go/postgres v0.0.0

replace github.com/EPOS-ERIC/epos-platform/packages/go/logging => /workspace/packages/go/logging
replace github.com/EPOS-ERIC/epos-platform/packages/go/migrations => /workspace/packages/go/migrations
replace github.com/EPOS-ERIC/epos-platform/packages/go/postgres => /workspace/packages/go/postgres

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/text v0.24.0 // indirect
)
