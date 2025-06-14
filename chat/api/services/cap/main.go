package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/DavidLee0620/GoIM/chat/foundation/web"
)

func main() {
	var log *logger.Logger
	traceIDFn := func(ctx context.Context) string {
		return web.GetTraceID(ctx).String() //TODO:需要从上下文中获取Trace IDs
	}
	//创建一个新的日志记录器，输出到标准输出，日志级别为Info，模块名为"CAP"，并传入traceID。
	log = logger.New(os.Stdout, logger.LevelInfo, "CAP", traceIDFn)
	//-----------------------------------------------------
	//创建一个空的上下文，作为程序运行的根上下文。
	ctx := context.Background()
	if err := run(ctx, log); err != nil {
		//记录错误日志，标签为"startup"，附带错误信息。
		log.Error(ctx, "startup", "err", err)
		//程序异常退出，返回状态码1。
		os.Exit(1)
	}
}

// 定义run函数，接收上下文和日志记录器，返回错误。
func run(ctx context.Context, log *logger.Logger) error {
	//记录信息日志，标签为"startup"，输出当前Go程序的最大CPU使用数。
	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))
	//-----------------------------------------------------
	log.Info(ctx, "startup", "status", "starting")
	defer log.Info(ctx, "startup", "status", "shutting down")
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM) //监听SIGINT和SIGTERM信号
	/*
		SIGINT（中断信号）和SIGTERM（终止信号）是Unix/Linux系统中常见的两种信号，用于通知进程进行中断或终止操作。

		SIGINT信号：
		通常由用户通过键盘输入Ctrl+C触发。
		发送给前台进程，表示用户希望中断当前程序。
		程序接收到SIGINT后，可以选择捕获该信号进行清理操作，或者默认终止。
		SIGTERM信号：
		是一种请求程序终止的信号，通常由系统或其他进程发送。
		与SIGINT不同，SIGTERM更“温和”，允许程序有机会优雅地关闭资源。
		程序可以捕获该信号，执行必要的清理后退出。
	*/
	<-shutdown
	return nil

}
