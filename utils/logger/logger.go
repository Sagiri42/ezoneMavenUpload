package logger

import (
	"log"
	"log/slog"
	"os"
)

var Logger *slog.Logger

func init() {
	var file *os.File
	var err error
	// 创建一个文件来保存日志
	if file, err = os.OpenFile("ezOneMavenUpload.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err != nil {
		log.Fatal("打开日志文件失败:", err)
	}
	//defer file.Close()

	// 配置 slog 使用文件作为日志输出
	Logger = slog.New(slog.NewTextHandler(file, nil))
}
