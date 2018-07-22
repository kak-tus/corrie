package applog

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"git.aqq.me/go/app/appconf"
	"github.com/iph0/merger"
	mapstruct "github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a wrapper around zap sugared logger
type Logger struct {
	*zap.SugaredLogger
	close func()
}

type loggerConfig struct {
	Tag          string
	Level        string
	Output       string
	OutputConfig map[string]interface{}
	Format       string
	FormatConfig *formatConfig
}

type fileOutputConfig struct {
	FilePath string
}

type formatConfig struct {
	Colors           bool
	TimestampFormat  string
	DisableTimestamp bool
}

const (
	pkgNameSep  = "/"
	funcNameSep = '.'
)

var defaultLoggerConfig = loggerConfig{
	Tag:    "app",
	Level:  "info",
	Output: "stderr",
	Format: "console",

	FormatConfig: &formatConfig{
		Colors:           false,
		TimestampFormat:  "2006-01-02T15:04:05.99999Z0700",
		DisableTimestamp: false,
	},
}

// NewLogger method creates new logger instance
func NewLogger() (*Logger, error) {
	var config loggerConfig
	configRaw := appconf.GetConfig()
	err := mapstruct.Decode(configRaw["log"], &config)

	if err != nil {
		return nil, fmt.Errorf("%s: invalid configuration: %s", errPref, err)
	}

	config = merger.Merge(defaultLoggerConfig, config).(loggerConfig)

	atomLevel := zap.NewAtomicLevel()
	err = atomLevel.UnmarshalText([]byte(config.Level))

	if err != nil {
		return nil, err
	}

	encoder, err := newEncoder(
		config.Format,
		config.FormatConfig,
	)

	if err != nil {
		return nil, err
	}

	var core zapcore.Core
	var close func()

	if config.Output == "file" {
		outputOpts := fileOutputConfig{}
		err := mapstruct.Decode(config.OutputConfig, &outputOpts)

		if err != nil {
			return nil, err
		}

		if outputOpts.FilePath == "" {
			return nil, fmt.Errorf("%s: file path not specified", errPref)
		}

		var ws zapcore.WriteSyncer
		ws, close, err = zap.Open(outputOpts.FilePath)

		if err != nil {
			return nil, err
		}

		core = zapcore.NewCore(encoder, ws, atomLevel)
	} else if config.Output == "stdsep" {
		highLevel := zap.LevelEnablerFunc(
			func(level zapcore.Level) bool {
				return level >= zapcore.ErrorLevel && level >= atomLevel.Level()
			},
		)

		lowLevel := zap.LevelEnablerFunc(
			func(level zapcore.Level) bool {
				return level < zapcore.ErrorLevel && level >= atomLevel.Level()
			},
		)

		highWS := zapcore.Lock(os.Stderr)
		lowWS := zapcore.Lock(os.Stdout)

		core = zapcore.NewTee(
			zapcore.NewCore(encoder, highWS, highLevel),
			zapcore.NewCore(encoder, lowWS, lowLevel),
		)
	} else if config.Output == "stdout" {
		ws := zapcore.Lock(os.Stdout)
		core = zapcore.NewCore(encoder, ws, atomLevel)
	} else { // stderr
		ws := zapcore.Lock(os.Stderr)
		core = zapcore.NewCore(encoder, ws, atomLevel)
	}

	// NOTE add additional outputs here...

	errWS := zapcore.Lock(os.Stderr)
	zapLogger := zap.New(core, zap.AddCaller(), zap.ErrorOutput(errWS))
	zapLogger = zapLogger.Named(config.Tag)

	zapLogger = zapLogger.With(
		zap.Int("pid", os.Getpid()),
	)

	return &Logger{
		SugaredLogger: zapLogger.Sugar(),
		close:         close,
	}, nil
}

// Close method performs correct closure of the logger.
func (*Logger) Close() {
	logger.Sync()

	if logger.close != nil {
		logger.close()
	}
}

func newEncoder(format string, config *formatConfig) (zapcore.Encoder, error) {
	var levelEncoder zapcore.LevelEncoder

	if config.Colors {
		levelEncoder = zapcore.CapitalColorLevelEncoder
	} else {
		levelEncoder = zapcore.CapitalLevelEncoder
	}

	var timeKey string
	var timeEncoder zapcore.TimeEncoder

	if !config.DisableTimestamp {
		timeKey = "time"
		timeEncoder = makeTimeEncoder(config.TimestampFormat)
	}

	encConfig := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        timeKey,
		NameKey:        "tag",
		CallerKey:      "caller",
		LineEnding:     "\n",
		EncodeLevel:    levelEncoder,
		EncodeTime:     timeEncoder,
		EncodeDuration: zapcore.NanosDurationEncoder,
		EncodeCaller:   callerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	if format == "json" {
		return zapcore.NewJSONEncoder(encConfig), nil
	}

	// NOTE add additional encoders here...

	return zapcore.NewConsoleEncoder(encConfig), nil
}

func makeTimeEncoder(format string) zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format(format))
	}
}

func callerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	longFuncName := runtime.FuncForPC(caller.PC).Name()

	tokens := strings.Split(longFuncName, pkgNameSep)
	tokensLen := len(tokens)
	shortFuncName := tokens[tokensLen-1]
	tokens = tokens[:tokensLen-1]

	var shortPkgName string

	if sepIdx := strings.IndexByte(shortFuncName, funcNameSep); sepIdx >= 0 {
		shortPkgName = string([]byte(shortFuncName)[:sepIdx])
	}

	tokens = append(tokens, shortPkgName)
	longPkgName := strings.Join(tokens, pkgNameSep)

	enc.AppendString(longPkgName)
}
