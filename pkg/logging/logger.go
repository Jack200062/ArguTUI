package logging

import (
	"fmt"

	"go.uber.org/zap"
)

type Logger struct {
	zapLogger *zap.SugaredLogger
}

func NewLogger() *Logger {
	config := zap.NewProductionConfig()
	config.DisableStacktrace = true

	logger, _ := config.Build()
	defer logger.Sync()

	return &Logger{zapLogger: logger.Sugar()}
}

func (l *Logger) Info(args ...interface{}) {
	l.zapLogger.Info(args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.zapLogger.Error(args...)
}

func (l *Logger) Debug(args ...interface{}) {
	l.zapLogger.Debug(args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.zapLogger.Warn(args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.zapLogger.Fatal(args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.zapLogger.Infof(format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	l.zapLogger.Error(err.Error())
	return err
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.zapLogger.Debugf(format, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.zapLogger.Warnf(format, args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.zapLogger.Fatalf(format, args...)
}
