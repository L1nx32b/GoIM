package zaplog

// 配置日志格式和日志输出方式
import (
	"GoChatServer/internal/config"
	"os"
	"path"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger
var logPath string

// 初始化
func init() {
	encoderConfig := zap.NewProductionEncoderConfig()
	// 设置日志记录的时间格式
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// 日志encoder还是JSONEncode, 把日志行格式化成JSON格式
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	conf := config.GetConfig()
	logPath = conf.LogPath
	// 创建了一个 MultiWriteSyncer (Tee)，目的是让日志既打印在屏幕（Stdout），又写进文件。
	file, _ := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 644)
	fileWriteSyncer := zapcore.AddSync(file)
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
		zapcore.NewCore(encoder, fileWriteSyncer, zapcore.DebugLevel),
	)
	logger = zap.New(core)
}

// 获取调用方的日志信息，包括函数名，文件名，行号
func getCallerInfoForLog() (callerFields []zap.Field) {
	pc, file, line, ok := runtime.Caller(2) // 回溯两层，拿到日志的调用方的函数信息
	if !ok {
		return
	}
	funcName := runtime.FuncForPC(pc).Name()
	funcName = path.Base(funcName) // Base函数返回路径的最后一个元素，只保留函数名

	callerFields = append(callerFields, zap.String("func", funcName), zap.String("file", file), zap.Int("line", line))
	return
}

func Info(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Info(message, fields...)
}

func Warn(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Warn(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Error(message, fields...)
}

func Fatal(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Fatal(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Debug(message, fields...)
}
