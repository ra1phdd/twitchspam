package logger

import "fmt"

type PrefixedLogger struct {
	inner  Logger
	prefix string
}

func NewPrefixedLogger(inner Logger, prefix string) *PrefixedLogger {
	return &PrefixedLogger{
		inner:  inner,
		prefix: prefix,
	}
}

func (p *PrefixedLogger) prefixed(msg string) string {
	return fmt.Sprintf("[%s] %s", p.prefix, msg)
}

func (p *PrefixedLogger) SetLogLevel(levelStr string) {
	p.inner.SetLogLevel(levelStr)
}

func (p *PrefixedLogger) GetLogLevel() string {
	return p.inner.GetLogLevel()
}

func (p *PrefixedLogger) Trace(msg string, args ...any) {
	p.inner.Trace(p.prefixed(msg), args...)
}

func (p *PrefixedLogger) Debug(msg string, args ...any) {
	p.inner.Debug(p.prefixed(msg), args...)
}

func (p *PrefixedLogger) Info(msg string, args ...any) {
	p.inner.Info(p.prefixed(msg), args...)
}

func (p *PrefixedLogger) Warn(msg string, args ...any) {
	p.inner.Warn(p.prefixed(msg), args...)
}

func (p *PrefixedLogger) Error(msg string, err error, args ...any) {
	p.inner.Error(p.prefixed(msg), err, args...)
}

func (p *PrefixedLogger) Fatal(msg string, err error, args ...any) {
	p.inner.Fatal(p.prefixed(msg), err, args...)
}
