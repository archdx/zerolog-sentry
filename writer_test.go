package zlogsentry

import (
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var logEventJSON = []byte(`{"level":"error","requestId":"bee07485-2485-4f64-99e1-d10165884ca7","error":"dial timeout","time":"2020-06-25T17:19:00+03:00","message":"test message"}`)

func TestParseLogEvent(t *testing.T) {
	ts := time.Now()

	now = func() time.Time { return ts }

	w, err := New("")
	require.Nil(t, err)

	ev, ok := w.parseLogEvent(logEventJSON)
	require.True(t, ok)
	zLevel, err := w.parseLogLevel(logEventJSON)
	assert.Nil(t, err)
	ev.Level = levelsMapping[zLevel]

	assert.Equal(t, ts, ev.Timestamp)
	assert.Equal(t, sentry.LevelError, ev.Level)
	assert.Equal(t, "zerolog", ev.Logger)
	assert.Equal(t, "test message", ev.Message)

	require.Len(t, ev.Exception, 1)
	assert.Equal(t, "dial timeout", ev.Exception[0].Value)

	require.Len(t, ev.Extra, 1)
	assert.Equal(t, "bee07485-2485-4f64-99e1-d10165884ca7", ev.Extra["requestId"])
}

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

func BenchmarkWriteLogEvent(b *testing.B) {
	w, err := New("")
	if err != nil {
		b.Errorf("failed to create writer: %v", err)
	}

	for i := 0; i < b.N; i++ {
		_, _ = w.Write(logEventJSON)
	}
}

func BenchmarkWriteLogEvent_Disabled(b *testing.B) {
	w, err := New("", WithLevels(zerolog.FatalLevel))
	if err != nil {
		b.Errorf("failed to create writer: %v", err)
	}

	for i := 0; i < b.N; i++ {
		_, _ = w.Write(logEventJSON)
	}
}

func BenchmarkWriteLogLevelEvent(b *testing.B) {
	w, err := New("")
	if err != nil {
		b.Errorf("failed to create writer: %v", err)
	}

	for i := 0; i < b.N; i++ {
		_, _ = w.WriteLevel(zerolog.ErrorLevel, logEventJSON)
	}
}

func BenchmarkWriteLogLevelEvent_DisabledLevel(b *testing.B) {
	w, err := New("", WithLevels(zerolog.FatalLevel))
	if err != nil {
		b.Errorf("failed to create writer: %v", err)
	}

	for i := 0; i < b.N; i++ {
		_, _ = w.WriteLevel(zerolog.ErrorLevel, logEventJSON)
	}
}
