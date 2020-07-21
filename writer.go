package zlogsentry

import (
	"io"
	"reflect"
	"time"
	"unsafe"

	"github.com/buger/jsonparser"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

var levelsMapping = map[zerolog.Level]sentry.Level{
	zerolog.DebugLevel: sentry.LevelDebug,
	zerolog.InfoLevel:  sentry.LevelInfo,
	zerolog.WarnLevel:  sentry.LevelWarning,
	zerolog.ErrorLevel: sentry.LevelError,
	zerolog.FatalLevel: sentry.LevelFatal,
	zerolog.PanicLevel: sentry.LevelFatal,
}

var _ = io.WriteCloser(new(Writer))

type Writer struct {
	client *sentry.Client

	levels       map[zerolog.Level]struct{}
	flushTimeout time.Duration
}

func (w *Writer) Write(data []byte) (int, error) {
	event, ok := w.parseLogEvent(data)
	if ok {
		w.client.CaptureEvent(event, nil, nil)
		// should flush before os.Exit
		if event.Level == sentry.LevelFatal {
			w.client.Flush(w.flushTimeout)
		}
	}

	return len(data), nil
}

func (w *Writer) Close() error {
	w.client.Flush(w.flushTimeout)
	return nil
}

func (w *Writer) parseLogEvent(data []byte) (*sentry.Event, bool) {
	const logger = "zerolog"

	lvlStr, err := jsonparser.GetUnsafeString(data, zerolog.LevelFieldName)
	if err != nil {
		return nil, false
	}

	lvl, err := zerolog.ParseLevel(lvlStr)
	if err != nil {
		return nil, false
	}

	_, enabled := w.levels[lvl]
	if !enabled {
		return nil, false
	}

	sentryLvl, ok := levelsMapping[lvl]
	if !ok {
		return nil, false
	}

	event := sentry.Event{
		Timestamp: time.Now().UTC(),
		Level:     sentryLvl,
		Logger:    logger,
	}

	err = jsonparser.ObjectEach(data, func(key, value []byte, vt jsonparser.ValueType, offset int) error {
		switch string(key) {
		// case zerolog.LevelFieldName, zerolog.TimestampFieldName:
		case zerolog.MessageFieldName:
			event.Message = bytesToStrUnsafe(value)
		case zerolog.ErrorFieldName:
			event.Exception = append(event.Exception, sentry.Exception{
				Value:      bytesToStrUnsafe(value),
				Stacktrace: newStacktrace(),
			})
		}

		return nil
	})

	if err != nil {
		return nil, false
	}

	return &event, true
}

func newStacktrace() *sentry.Stacktrace {
	const (
		currentModule = "github.com/archdx/zerolog-sentry"
		zerologModule = "github.com/rs/zerolog"
	)

	st := sentry.NewStacktrace()

	threshold := len(st.Frames) - 1
	// drop current module frames
	for ; threshold > 0 && st.Frames[threshold].Module == currentModule; threshold-- {
	}

outer:
	// try to drop zerolog module frames after logger call point
	for i := threshold; i > 0; i-- {
		if st.Frames[i].Module == zerologModule {
			for j := i - 1; j >= 0; j-- {
				if st.Frames[j].Module != zerologModule {
					threshold = j
					break outer
				}
			}

			break
		}
	}

	st.Frames = st.Frames[:threshold+1]

	return st
}

func bytesToStrUnsafe(data []byte) string {
	h := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	return *(*string)(unsafe.Pointer(&reflect.StringHeader{Data: h.Data, Len: h.Len}))
}

type WriterOption interface {
	apply(*config)
}

type optionFunc func(*config)

func (fn optionFunc) apply(c *config) { fn(c) }

type config struct {
	levels       []zerolog.Level
	sampleRate   float64
	release      string
	environment  string
	serverName   string
	debug        bool
	flushTimeout time.Duration
}

// WithLevels configures zerolog levels that have to be sent to Sentry. Default levels are error, fatal, panic
func WithLevels(levels ...zerolog.Level) WriterOption {
	return optionFunc(func(cfg *config) {
		cfg.levels = levels
	})
}

// WithSampleRate configures the sample rate as a percentage of events to be sent in the range of 0.0 to 1.0
func WithSampleRate(rate float64) WriterOption {
	return optionFunc(func(cfg *config) {
		cfg.sampleRate = rate
	})
}

func WithRelease(release string) WriterOption {
	return optionFunc(func(cfg *config) {
		cfg.release = release
	})
}

func WithEnvironment(environment string) WriterOption {
	return optionFunc(func(cfg *config) {
		cfg.environment = environment
	})
}

// WithServerName configures the server name field for events. Default value is OS hostname
func WithServerName(serverName string) WriterOption {
	return optionFunc(func(cfg *config) {
		cfg.serverName = serverName
	})
}

// WithDebug enables sentry client debug logs
func WithDebug() WriterOption {
	return optionFunc(func(cfg *config) {
		cfg.debug = true
	})
}

func New(dsn string, opts ...WriterOption) (*Writer, error) {
	cfg := newDefaultConfig()
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:         dsn,
		SampleRate:  cfg.sampleRate,
		Release:     cfg.release,
		Environment: cfg.environment,
		ServerName:  cfg.serverName,
		Debug:       cfg.debug,
	})

	if err != nil {
		return nil, err
	}

	levels := make(map[zerolog.Level]struct{}, len(cfg.levels))
	for _, lvl := range cfg.levels {
		levels[lvl] = struct{}{}
	}

	return &Writer{
		client:       client,
		levels:       levels,
		flushTimeout: cfg.flushTimeout,
	}, nil
}

func newDefaultConfig() config {
	return config{
		levels: []zerolog.Level{
			zerolog.ErrorLevel,
			zerolog.FatalLevel,
			zerolog.PanicLevel,
		},
		sampleRate:   1.0,
		flushTimeout: 3 * time.Second,
	}
}
