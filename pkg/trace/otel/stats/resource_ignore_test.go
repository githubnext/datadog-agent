// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package stats

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/DataDog/datadog-agent/pkg/opentelemetry-mapping-go/otlp/attributes"
	"github.com/DataDog/datadog-agent/pkg/trace/config"
)

func TestOTLPResourceIgnoreNamingVersions(t *testing.T) {
	translator, err := attributes.NewTranslator(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	for _, version := range []string{"v1", "v2"} {
		t.Run(version, func(t *testing.T) {
			for _, tc := range []struct {
				name    string
				ignore  []string
				dropped bool
			}{
				{name: "unset"},
				{name: "empty", ignore: []string{}},
				{name: "matching", ignore: []string{"GET /users"}, dropped: true},
				{name: "nonmatching", ignore: []string{"POST /users"}},
				{name: "multiple matching", ignore: []string{"POST /users", "GET /users"}, dropped: true},
				{name: "multiple nonmatching", ignore: []string{"POST /users", "GET /health"}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					traces := ptrace.NewTraces()
					resourceSpans := traces.ResourceSpans().AppendEmpty()
					resourceSpans.Resource().Attributes().PutStr("service.name", "test-service")
					span := resourceSpans.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
					span.SetTraceID(pcommon.TraceID{15: 1})
					span.SetSpanID(pcommon.SpanID{7: 1})
					span.SetName("request")
					span.SetKind(ptrace.SpanKindServer)
					span.SetStartTimestamp(1)
					span.SetEndTimestamp(2)
					span.Attributes().PutStr("http.request.method", "GET")
					span.Attributes().PutStr("http.route", "/users")

					conf := config.New()
					conf.OTLPReceiver.AttributesTranslator = translator
					conf.Ignore["resource"] = tc.ignore
					if version == "v1" {
						conf.Features["disable_operation_and_resource_name_logic_v2"] = struct{}{}
					}
					inputs := OTLPTracesToConcentratorInputs(traces, conf, nil, nil, nil)
					if tc.dropped {
						require.Empty(t, inputs)
						return
					}
					require.Len(t, inputs, 1)
					require.Len(t, inputs[0].Traces, 1)
					require.Len(t, inputs[0].Traces[0].TraceChunk.Spans, 1)
					result := inputs[0].Traces[0].TraceChunk.Spans[0]
					require.Equal(t, "GET /users", result.Resource)
					require.Equal(t, "test-service", result.Service)
					require.Equal(t, uint64(1), result.TraceID)
					require.Equal(t, uint64(1), result.SpanID)
					require.Equal(t, int64(1), result.Start)
					require.Equal(t, int64(1), result.Duration)
				})
			}
		})
	}
}
