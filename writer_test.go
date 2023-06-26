package zlogsentry

import (
	"errors"
	"io"
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

func TestParseLogLevel(t *testing.T) {
	w, err := New("")
	require.Nil(t, err)

	level, err := w.parseLogLevel(logEventJSON)
	require.Nil(t, err)
	assert.Equal(t, zerolog.ErrorLevel, level)
}

func TestWrite(t *testing.T) {
	beforeSendCalled := false
	writer, err := New("", WithBeforeSend(func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
		assert.Equal(t, sentry.LevelError, event.Level)
		assert.Equal(t, "test message", event.Message)
		require.Len(t, event.Exception, 1)
		assert.Equal(t, "dial timeout", event.Exception[0].Value)
		assert.True(t, time.Since(event.Timestamp).Minutes() < 1)
		assert.Equal(t, "bee07485-2485-4f64-99e1-d10165884ca7", event.Extra["requestId"])
		beforeSendCalled = true
		return event
	}))
	require.Nil(t, err)

	var zerologError error
	zerolog.ErrorHandler = func(err error) {
		zerologError = err
	}

	// use io.MultiWriter to enforce using the Write() method
	log := zerolog.New(io.MultiWriter(writer)).With().Timestamp().
		Str("requestId", "bee07485-2485-4f64-99e1-d10165884ca7").
		Logger()
	log.Err(errors.New("dial timeout")).
		Msg("test message")

	require.Nil(t, zerologError)
	require.True(t, beforeSendCalled)
}

func TestWriteLevel(t *testing.T) {
	beforeSendCalled := false
	writer, err := New("", WithBeforeSend(func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
		assert.Equal(t, sentry.LevelError, event.Level)
		assert.Equal(t, "test message", event.Message)
		require.Len(t, event.Exception, 1)
		assert.Equal(t, "dial timeout", event.Exception[0].Value)
		assert.True(t, time.Since(event.Timestamp).Minutes() < 1)
		assert.Equal(t, "bee07485-2485-4f64-99e1-d10165884ca7", event.Extra["requestId"])
		beforeSendCalled = true
		return event
	}))
	require.Nil(t, err)

	var zerologError error
	zerolog.ErrorHandler = func(err error) {
		zerologError = err
	}

	log := zerolog.New(writer).With().Timestamp().
		Str("requestId", "bee07485-2485-4f64-99e1-d10165884ca7").
		Logger()
	log.Err(errors.New("dial timeout")).
		Msg("test message")

	require.Nil(t, zerologError)
	require.True(t, beforeSendCalled)
}

func TestWrite_Disabled(t *testing.T) {
	beforeSendCalled := false
	writer, err := New("",
		WithLevels(zerolog.FatalLevel),
		WithBeforeSend(func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			beforeSendCalled = true
			return event
		}))
	require.Nil(t, err)

	var zerologError error
	zerolog.ErrorHandler = func(err error) {
		zerologError = err
	}

	// use io.MultiWriter to enforce using the Write() method
	log := zerolog.New(io.MultiWriter(writer)).With().Timestamp().
		Str("requestId", "bee07485-2485-4f64-99e1-d10165884ca7").
		Logger()
	log.Err(errors.New("dial timeout")).
		Msg("test message")

	require.Nil(t, zerologError)
	require.False(t, beforeSendCalled)
}

func TestWriteLevel_Disabled(t *testing.T) {
	beforeSendCalled := false
	writer, err := New("",
		WithLevels(zerolog.FatalLevel),
		WithBeforeSend(func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			beforeSendCalled = true
			return event
		}))
	require.Nil(t, err)

	var zerologError error
	zerolog.ErrorHandler = func(err error) {
		zerologError = err
	}

	log := zerolog.New(writer).With().Timestamp().
		Str("requestId", "bee07485-2485-4f64-99e1-d10165884ca7").
		Logger()
	log.Err(errors.New("dial timeout")).
		Msg("test message")

	require.Nil(t, zerologError)
	require.False(t, beforeSendCalled)
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

func BenchmarkParseLogEvent_Disabled(b *testing.B) {
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

func BenchmarkWriteLogLevelEvent_Disabled(b *testing.B) {
	w, err := New("", WithLevels(zerolog.FatalLevel))
	if err != nil {
		b.Errorf("failed to create writer: %v", err)
	}

	for i := 0; i < b.N; i++ {
		_, _ = w.WriteLevel(zerolog.ErrorLevel, logEventJSON)
	}
}
