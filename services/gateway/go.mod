module github.com/EPOS-ERIC/epos-platform/services/gateway

go 1.27

require (
	github.com/EPOS-ERIC/epos-platform/packages/go/apidocs v0.0.0
	github.com/EPOS-ERIC/epos-platform/packages/go/logging v0.0.0
	github.com/EPOS-ERIC/epos-platform/packages/go/middleware v0.0.0
	github.com/EPOS-ERIC/epos-platform/packages/go/observability v0.0.0
	github.com/EPOS-ERIC/epos-platform/packages/protobuf v0.0.0
	github.com/go-chi/chi/v5 v5.2.5
	github.com/go-faster/errors v0.8.0
	github.com/go-faster/jx v1.2.0
	github.com/google/uuid v1.6.0
	github.com/ogen-go/ogen v1.24.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.71.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0
	go.opentelemetry.io/otel v1.46.0
	go.opentelemetry.io/otel/trace v1.46.0
	go.yaml.in/yaml/v3 v3.0.5
	google.golang.org/grpc v1.83.2
	google.golang.org/protobuf v1.36.12
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.12-20260825204119-511051f7f437.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dlclark/regexp2 v1.12.0 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-faster/yaml v0.4.6 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/bridges/otelslog v0.19.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0 // indirect
	go.opentelemetry.io/otel/log v0.20.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/sdk v1.46.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.46.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/exp v0.0.0-20230725093048-515e97ebf090 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260825221802-da73d73af1c5 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/EPOS-ERIC/epos-platform/packages/go/logging => ../../packages/go/logging

replace github.com/EPOS-ERIC/epos-platform/packages/go/apidocs => ../../packages/go/apidocs

replace github.com/EPOS-ERIC/epos-platform/packages/go/middleware => ../../packages/go/middleware

replace github.com/EPOS-ERIC/epos-platform/packages/go/observability => ../../packages/go/observability

replace github.com/EPOS-ERIC/epos-platform/packages/protobuf => ../../packages/protobuf

tool github.com/ogen-go/ogen/cmd/ogen
