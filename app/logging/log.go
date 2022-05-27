package logging

import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"strings"
	"vsphere-facade/app/utils/intutils"
	"vsphere-facade/app/utils/stringutils"
	"vsphere-facade/config"
)

var log *zap.Logger

var settingLevel zapcore.Level

func Setup() {
	var coreArr []zapcore.Core
	//获取编码器
	encoderConfig := zap.NewProductionEncoderConfig()       //NewJSONEncoder()输出json格式，NewConsoleEncoder()输出普通文本格式
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   //指定时间格式
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder //按级别显示不同颜色，不需要的话取值zapcore.CapitalLevelEncoder就可以了
	if config.G.Server.Log.EnableFullPath {
		encoderConfig.EncodeCaller = zapcore.FullCallerEncoder //显示完整文件路径
	}
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	//日志级别
	level, err := zapcore.ParseLevel(config.G.Server.Log.Level)
	if err != nil {
		fmt.Printf("日志级别设置出错，使用默认级别: %s", zapcore.InfoLevel.String())
		level = zapcore.InfoLevel
	}
	settingLevel = level
	priority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool { //info和debug级别,debug级别是最低的
		return lev >= level
	})

	infoFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   stringutils.EPTThen(strings.TrimSuffix(config.G.Server.Log.Path, "/"), ".") + "/vsphere-facade.log", //日志文件存放目录，如果文件夹不存在会自动创建
		MaxSize:    intutils.ZeroThen(config.G.Server.Log.MaxSize, 1),                                                   //文件大小限制,单位MB
		MaxBackups: intutils.ZeroThen(config.G.Server.Log.MaxBackups, 10),                                               //最大保留日志文件数量
		MaxAge:     intutils.ZeroThen(config.G.Server.Log.MaxAge, 7),                                                    //日志文件保留天数
		Compress:   false,                                                                                               //是否压缩处理
	})
	infoFileCore := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(infoFileWriteSyncer, zapcore.AddSync(os.Stdout)), priority) //第三个及之后的参数为写入文件的日志级别,ErrorLevel模式只记录error级别的日志
	coreArr = append(coreArr, infoFileCore)
	log = zap.New(zapcore.NewTee(coreArr...), zap.AddCaller()) //zap.AddCaller()为显示文件名和行号，可省略
}

func L() *zap.SugaredLogger {
	return log.Sugar()
}

func IsDebug() bool {
	return settingLevel == zapcore.DebugLevel
}

func Sync() {
	err := log.Sync()
	if err != nil {
		fmt.Println(err)
		return
	}
}
