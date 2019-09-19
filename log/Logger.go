package log

import (
	"github.com/transsnet/vlog/log/kafka"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"strings"
)

var (
	logInfo   *zap.SugaredLogger
	logErr    *zap.SugaredLogger
	logAccess *zap.SugaredLogger
)

type LoggerConfig struct {
	EnableKafkaLogger bool
	*BaseLoggerConfig
	*KafkaLoggerConfig
}

type BaseLoggerConfig struct {
	LogPath     string
	Mode        string
	ServiceName string
}

type KafkaLoggerConfig struct {
	KafkaClient *kafka.KafkaClient
	InfoTopic   string
	ErrorTopic  string
}

func InitLog(config *LoggerConfig) {

	if err := MakeDir(config.LogPath); err != nil {
		panic(err)
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	// 增加几个初始字段
	additionalFields := initAdditionFields(config)

	// 初始化kafka
	if config.EnableKafkaLogger {
		kafka.InitKafkaClient(config.KafkaClient)
	}

	var core zapcore.Core

	// log info
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   strings.Join([]string{config.LogPath, "info.log"}, "/"),
		MaxSize:    500, // megabytes
		MaxBackups: 30,
		MaxAge:     30, // days
		LocalTime:  false,
	})

	if config.EnableKafkaLogger {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.NewMultiWriteSyncer(kafka.New(config.InfoTopic), w),
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

	//log error
	w = zapcore.AddSync(&lumberjack.Logger{
		Filename:   strings.Join([]string{config.LogPath, "error.log"}, "/"),
		MaxSize:    500,
		MaxBackups: 30,
		MaxAge:     30, // days
		LocalTime:  false,
	})
	encoderCfg.CallerKey = "caller"

	if config.EnableKafkaLogger {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.NewMultiWriteSyncer(kafka.New(config.ErrorTopic), w),
			zap.ErrorLevel,
		)
	} else {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			w,
			zap.ErrorLevel,
		)
	}

	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), additionalFields)
	logErr = logger.Sugar()

	// access
	w = zapcore.AddSync(&lumberjack.Logger{
		Filename:   strings.Join([]string{config.LogPath, "access.log"}, "/"),
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

func initAdditionFields(config *LoggerConfig) zap.Option {
	additionFields := zap.Fields(zap.String("serviceName", config.ServiceName),
		zap.String("logPath", config.LogPath))
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
