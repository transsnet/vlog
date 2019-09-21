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

	// 配置一个 log info
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   strings.Join([]string{config.Base.LogPath, "info.log"}, "/"),
		MaxSize:    500, // megabytes
		MaxBackups: 30,
		MaxAge:     30, // days
		LocalTime:  false,
	})

	if config.EnableKafka {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.NewMultiWriteSyncer(kafka.New(config.Kafka.InfoTopic), w),
			zap.InfoLevel,
		)
	} else {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			w,
			zap.InfoLevel,
		)
	}

	logger := zap.New(core, additionalFields)
	logInfo = logger.Sugar()

	// 配置一个 log error
	w = zapcore.AddSync(&lumberjack.Logger{
		Filename:   strings.Join([]string{config.Base.LogPath, "error.log"}, "/"),
		MaxSize:    500,
		MaxBackups: 30,
		MaxAge:     30, // days
		LocalTime:  false,
	})
	encoderCfg.CallerKey = "caller"
	encoderCfg.StacktraceKey = "stacktrace"

	if config.EnableKafka {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.NewMultiWriteSyncer(kafka.New(config.Kafka.ErrorTopic), w),
			zap.ErrorLevel,
		)
	} else {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			w,
			zap.ErrorLevel,
		)
	}

	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), additionalFields, zap.AddStacktrace(zap.ErrorLevel))
	logErr = logger.Sugar()

	//  配置一个 access
	w = zapcore.AddSync(&lumberjack.Logger{
		Filename:   strings.Join([]string{config.Base.LogPath, "access.log"}, "/"),
		MaxSize:    500, // megabytes
		MaxBackups: 30,
		MaxAge:     30, // days
		LocalTime:  false,
	})

	core = zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		w,
		zap.InfoLevel,
	)
	logger = zap.New(core, additionalFields)
	logAccess = logger.Sugar()

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
