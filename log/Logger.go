package log

import (
	"github.com/transsnet/vlog/log/kafka"
	"github.com/transsnet/vlog/log/model"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"strings"
)

var (
	logInfo   *zap.SugaredLogger
	logErr    *zap.SugaredLogger
	logAccess *zap.SugaredLogger
	logDebug  *zap.SugaredLogger
)

// 初始化日志配置
func InitLog(config *model.LoggerConfig) {

	// 校验一下，报错就抛出
	validate(config)

	// 创建日志目录
	if err := MakeDir(config.Base.LogPath); err != nil {
		panic(err)
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	// 增加几个初始字段
	additionalFields := initAdditionFields(config)

	// 初始化kafka
	if config.EnableKafka {
		kafka.InitKafkaClient(config.Kafka.Client)
	}

	var core zapcore.Core
	var logLevel zapcore.Level

	switch config.Base.LogLevel {
	case "debug":
		logLevel = zap.DebugLevel
	case "info":
		logLevel = zap.InfoLevel
	case "error":
		logLevel = zap.ErrorLevel
	default:
		logLevel = zap.InfoLevel
	}

	//  配置一个 debug
	dw := zapcore.AddSync(&lumberjack.Logger{
		Filename:   strings.Join([]string{config.Base.LogPath, "debug.log"}, "/"),
		MaxSize:    500, // megabytes
		MaxBackups: 15,
		MaxAge:     7, // days
		LocalTime:  false,
	})

	core = zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		dw,
		logLevel,
	)
	dLogger := zap.New(core)
	logDebug = dLogger.Sugar()

	//  配置一个 access
	aw := zapcore.AddSync(&lumberjack.Logger{
		Filename:   strings.Join([]string{config.Base.LogPath, "access.log"}, "/"),
		MaxSize:    500, // megabytes
		MaxBackups: 15,
		MaxAge:     7, // days
		LocalTime:  false,
	})

	core = zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		aw,
		logLevel,
	)
	aLogger := zap.New(core)
	logAccess = aLogger.Sugar()

	// 配置一个 log info
	iw := zapcore.AddSync(&lumberjack.Logger{
		Filename:   strings.Join([]string{config.Base.LogPath, "info.log"}, "/"),
		MaxSize:    500, // megabytes
		MaxBackups: 15,
		MaxAge:     7, // days
		LocalTime:  false,
	})

	core = zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		iw,
		logLevel,
	)

	iLogger := zap.New(core)
	logInfo = iLogger.Sugar()

	// 配置一个 log error
	ew := zapcore.AddSync(&lumberjack.Logger{
		Filename:   strings.Join([]string{config.Base.LogPath, "error.log"}, "/"),
		MaxSize:    500,
		MaxBackups: 15,
		MaxAge:     7, // days
		LocalTime:  false,
	})
	encoderCfg.CallerKey = "caller"
	encoderCfg.StacktraceKey = "stacktrace"

	if config.EnableKafka {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.NewMultiWriteSyncer(kafka.New(config.Kafka.ErrorTopic, config.Kafka.Filter), ew),
			logLevel,
		)
	} else {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			ew,
			logLevel,
		)
	}

	eLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1+config.Base.CallerSkip), additionalFields,
		zap.AddStacktrace(zap.ErrorLevel))
	logErr = eLogger.Sugar()
}

// 初始化几个额外的字段
func initAdditionFields(config *model.LoggerConfig) zap.Option {
	additionFields := zap.Fields(zap.String("serviceName", config.Base.ServiceName),
		zap.String("logPath", config.Base.LogPath))
	return additionFields
}

func Error(args ...interface{}) {
	logErr.Error(args)
}

func Errorf(format string, args ...interface{}) {
	logErr.Errorf(format, args)
}

func Info(args ...interface{}) {
	logInfo.Info(args)
}

func Infof(format string, args ...interface{}) {
	logInfo.Infof(format, args...)
}

func Access(args ...interface{}) {
	logAccess.Info(args)
}

func Accessf(format string, args ...interface{}) {
	logAccess.Infof(format, args...)
}

func Debug(args ...interface{}) {
	logDebug.Info(args)
}

func Debugf(format string, args ...interface{}) {
	logDebug.Infof(format, args...)
}

func IsFileExist(f string) bool {
	_, err := os.Stat(f)
	return err == nil || os.IsExist(err)
}

func MakeDir(f string) error {
	if IsFileExist(f) {
		return nil
	}
	return os.MkdirAll(f, os.ModePerm)
}

// 对配置进行校验
func validate(config *model.LoggerConfig) {
	_validator := validator.New()
	if err := _validator.Struct(config); err != nil {
		panic(err)
	}
}
