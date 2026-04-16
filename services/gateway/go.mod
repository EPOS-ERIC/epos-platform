module github.com/EPOS-ERIC/epos-platform/services/gateway

go 1.26.2

require github.com/EPOS-ERIC/epos-platform/packages/go/logging v0.0.0

require (
	github.com/kr/pretty v0.2.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)

require (
	github.com/go-chi/chi/v5 v5.2.5
	go.yaml.in/yaml/v3 v3.0.3
)

replace github.com/EPOS-ERIC/epos-platform/packages/go/logging => ../../packages/go/logging
