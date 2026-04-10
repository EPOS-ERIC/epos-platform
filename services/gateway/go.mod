module github.com/EPOS-ERIC/epos-platform/orchestrator-service

go 1.26.2

require github.com/EPOS-ERIC/epos-platform/packages/go/logging v0.0.0

require github.com/caarlos0/env/v11 v11.4.0 // indirect

replace github.com/EPOS-ERIC/epos-platform/packages/go/logging => ../packages/go/logging
