package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"hxy352/src/model"
	"time"
)

func InitLogger(conf *model.ZapConfig) (err error) {
	cfg := zap.NewProductionConfig()
	cfg.Sampling = nil
	lvl := zapcore.InfoLevel
	_ = lvl.Set(conf.Level)

	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.Encoding = conf.Encoding
	cfg.OutputPaths = conf.OutputPaths
	cfg.ErrorOutputPaths = conf.ErrorOutputPaths
	cfg.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	cfg.EncoderConfig.EncodeTime = EpochMillisTimeEncoder

	logger, err := cfg.Build(zap.Fields(
		zap.String("server_name", "black_phone_storage"),
	))
	if err != nil {
		panic(err)
	}

	logger = logger.WithOptions(zap.AddCallerSkip(1)) // 跳过封装的一层调用栈
	zap.ReplaceGlobals(logger)

	return
}

func EpochMillisTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(time.RFC3339))
}

func Debug(args ...interface{}) {
	zap.S().Debug(args...)
}

func Debugf(template string, args ...interface{}) {
	zap.S().Debugf(template, args...)
}

func Info(args ...interface{}) {
	zap.S().Info(args...)
}

func Infof(template string, args ...interface{}) {
	zap.S().Infof(template, args...)
}

func Warn(args ...interface{}) {
	zap.S().Warn(args...)
}

func Warnf(template string, args ...interface{}) {
	zap.S().Warnf(template, args...)
}

func Error(args ...interface{}) {
	zap.S().Error(args...)
}

func Errorf(template string, args ...interface{}) {
	zap.S().Errorf(template, args...)
}

func Panic(args ...interface{}) {
	zap.S().Panic(args...)
}

func Panicf(template string, args ...interface{}) {
	zap.S().Panicf(template, args...)
}
