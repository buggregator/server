package sentry

import (
	"google.golang.org/protobuf/proto"

	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

func otlpSeverity(n int32) logspb.SeverityNumber { return logspb.SeverityNumber(n) }

// otlpTracesFromJSONForTest turns an OTLP/JSON traces body into protobuf.
func otlpTracesFromJSONForTest(body []byte) ([]byte, error) {
	var td tracepb.TracesData
	if err := decodeOTLP(body, true, &td); err != nil {
		return nil, err
	}
	return proto.Marshal(&td)
}
