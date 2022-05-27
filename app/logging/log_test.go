package logging

import (
	"errors"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"testing"
)

var log99 *zap.Logger

func TestLog(t *testing.T) {
	var coreArr []zapcore.Core

	//获取编码器
	encoderConfig := zap.NewProductionEncoderConfig()       //NewJSONEncoder()输出json格式，NewConsoleEncoder()输出普通文本格式
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   //指定时间格式
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder //按级别显示不同颜色，不需要的话取值zapcore.CapitalLevelEncoder就可以了
	encoderConfig.EncodeCaller = zapcore.FullCallerEncoder  //显示完整文件路径
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	//日志级别
	priority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool { //info和debug级别,debug级别是最低的
		return lev >= zap.DebugLevel
	})

	//info文件writeSyncer
	infoFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "./vsphere_api.log", //日志文件存放目录，如果文件夹不存在会自动创建
		MaxSize:    2,                   //文件大小限制,单位MB
		MaxBackups: 100,                 //最大保留日志文件数量
		MaxAge:     30,                  //日志文件保留天数
		Compress:   false,               //是否压缩处理
	})
	infoFileCore := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(infoFileWriteSyncer, zapcore.AddSync(os.Stdout)), priority) //第三个及之后的参数为写入文件的日志级别,ErrorLevel模式只记录error级别的日志
	coreArr = append(coreArr, infoFileCore)
	log99 = zap.New(zapcore.NewTee(coreArr...), zap.AddCaller()) //zap.AddCaller()为显示文件名和行号，可省略

	log99.Sugar().Debugf("hello debug %d %s %s", 1, "test", errors.New("asdddd"))
	log99.Sugar().Infof("hello info")
	log99.Sugar().Warnf("hello warn")
	log99.Sugar().Errorf("hello error")
	log99.Sugar().Fatalf("hello fatal")
	log99.Sugar().Panicf("hello panic")
}
