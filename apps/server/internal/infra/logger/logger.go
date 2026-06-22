package logger

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"github.com/xuanye/one-round/apps/server/internal/config"
)

type Logger interface {
	Info(message string, fields ...zap.Field)
	Debug(message string, fields ...zap.Field)
	Error(message string, fields ...zap.Field)
	Warn(message string, fields ...zap.Field)
	Fatal(message string, fields ...zap.Field)
	InfoF(message string, a ...any)
	DebugF(message string, a ...any)
	ErrorF(message string, a ...any)
	WarnF(message string, a ...any)
	FatalF(message string, a ...any)
	Sync() error
}

type zapLoggerAdapter struct {
	logger *zap.Logger
}

func (log *zapLoggerAdapter) Info(message string, fields ...zap.Field) {
	log.logger.Info(message, fields...)
}

func (log *zapLoggerAdapter) Debug(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	log.logger.Debug(message, fields...)
}

func (log *zapLoggerAdapter) Error(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	log.logger.Error(message, fields...)
}

func (log *zapLoggerAdapter) Warn(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	log.logger.Warn(message, fields...)
}

func (log *zapLoggerAdapter) Fatal(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	log.logger.Fatal(message, fields...)
}

func (log *zapLoggerAdapter) InfoF(message string, a ...any) {
	log.logger.Info(fmt.Sprintf(message, a...))
}

func (log *zapLoggerAdapter) DebugF(message string, a ...any) {
	log.logger.Debug(fmt.Sprintf(message, a...))
}

func (log *zapLoggerAdapter) ErrorF(message string, a ...any) {
	log.logger.Error(fmt.Sprintf(message, a...))
}

func (log *zapLoggerAdapter) WarnF(message string, a ...any) {
	log.logger.Warn(fmt.Sprintf(message, a...))
}

func (log *zapLoggerAdapter) FatalF(message string, a ...any) {
	log.logger.Fatal(fmt.Sprintf(message, a...))
}

func (log *zapLoggerAdapter) Sync() error {
	return log.logger.Sync()
}

func getCallerInfoForLog() (callerFields []zap.Field) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return
	}
	funcName := runtime.FuncForPC(pc).Name()
	funcName = path.Base(funcName)
	callerFields = append(callerFields, zap.String("func", funcName), zap.String("file", file), zap.Int("line", line))
	return
}

func getFileLogWriter(logFile string) (writeSyncer zapcore.WriteSyncer) {
	today := time.Now().Format("2006-01-02")
	datedFile := fmt.Sprintf("%s-%s.log",
		strings.TrimSuffix(logFile, ".log"),
		today,
	)
	lumberJackLogger := &lumberjack.Logger{
		Filename:   datedFile,
		MaxSize:    100,
		MaxBackups: 60,
		MaxAge:     60,
		Compress:   false,
		LocalTime:  true,
	}
	return zapcore.AddSync(lumberJackLogger)
}

func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func NewZapLoggerAdapter(config *config.Config) Logger {
	// Ensure the log directory exists before lumberjack tries to write.
	if dir := path.Dir(config.Log.OutputPath); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	fileWriteSyncer := getFileLogWriter(config.Log.OutputPath)
	zapLevel := parseLogLevel(config.Log.Level)
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapLevel),
		zapcore.NewCore(encoder, fileWriteSyncer, zapLevel),
	)
	rawLogger := zap.New(core)
	return &zapLoggerAdapter{logger: rawLogger}
}

func NewConsole() Logger {
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
	return &zapLoggerAdapter{logger: zap.New(core)}
}

func NewNop() Logger {
	return &zapLoggerAdapter{logger: zap.NewNop()}
}
