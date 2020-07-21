package zlogsentry

import (
	"testing"

	"github.com/rs/zerolog"
)

var logEventJSON = []byte(`{"level":"error","requestId":"bee07485-2485-4f64-99e1-d10165884ca7","error":"dial timeout","time":"2020-06-25T17:19:00+03:00","message":"test message"}`)

func BenchmarkParseLogEvent(b *testing.B) {
	w, err := New("")
	if err != nil {
		b.Errorf("failed to create writer: %v", err)
	}

	for i := 0; i < b.N; i++ {
		w.parseLogEvent(logEventJSON)
	}
}

func BenchmarkParseLogEvent_DisabledLevel(b *testing.B) {
	w, err := New("", WithLevels(zerolog.FatalLevel))
	if err != nil {
		b.Errorf("failed to create writer: %v", err)
	}

	for i := 0; i < b.N; i++ {
		w.parseLogEvent(logEventJSON)
	}
}
