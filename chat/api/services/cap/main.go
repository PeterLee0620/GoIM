package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/DavidLee0620/GoIM/chat/app/sdk/chat"
	"github.com/DavidLee0620/GoIM/chat/app/sdk/chat/users"
	"github.com/DavidLee0620/GoIM/chat/app/sdk/mux"
	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/DavidLee0620/GoIM/chat/foundation/web"
	"github.com/ardanlabs/conf/v3"
	"github.com/nats-io/nats.go"
)

var build = "develop"

func main() {
	var log *logger.Logger

	traceIDFn := func(ctx context.Context) string {
		return web.GetTraceID(ctx).String()
	}

	log = logger.New(os.Stdout, logger.LevelInfo, "CAP", traceIDFn)

	// -------------------------------------------------------------------------

	ctx := context.Background()

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {

	// -------------------------------------------------------------------------
	// GOMAXPROCS

	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// -------------------------------------------------------------------------
	// Configuration

	cfg := struct {
		conf.Version
		Web struct {
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:10s"`
			IdleTimeout     time.Duration `conf:"default:120s"`
			ShutdownTimeout time.Duration `conf:"default:20s"`
			APIHost         string        `conf:"default:0.0.0.0:3000"`
		}
		NATS struct {
			Host    string `conf:"default:demo.nats.io"`
			Subject string `conf:"default:lee-cap"`
		}
	}{
		Version: conf.Version{
			Build: build,
			Desc:  "CAP",
		},
	}

	const prefix = "SALES"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	// -------------------------------------------------------------------------
	// App Starting

	log.Info(ctx, "starting service", "version", cfg.Build)
	defer log.Info(ctx, "shutdown complete")

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config for output: %w", err)
	}
	log.Info(ctx, "startup", "config", out)

	log.BuildInfo(ctx)
	// -------------------------------------------------------------------------
	// NATS Connection
	nc, err := nats.Connect(cfg.NATS.Host)
	if err != nil {
		return fmt.Errorf("nats create: %w", err)
	}
	defer nc.Close()
	chat, err := chat.New(log, nc, cfg.NATS.Subject, users.New(log))
	if err != nil {
		return fmt.Errorf("chat: %w", err)
	}
	defer chat.Shutdown(context.TODO())
	// -------------------------------------------------------------------------
	// Start API Service

	log.Info(ctx, "startup", "status", "initializing V1 API support")
	//优雅关闭 ：在这段代码中，它用于优雅地关闭服务器。当程序接收到 SIGINT（通常是用户按下 Ctrl+C 时发送的信号）或 SIGTERM（用于请求程序终止的信号）时，程序会捕获这些信号，并执行关闭服务器的操作，而不是直接强制退出，从而确保资源能够正确释放，正在进行的操作能够妥善处理。
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	cfgMux := mux.Config{
		Log:  log,
		Chat: chat,
	}

	webAPI := mux.WebAPI(cfgMux)

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      webAPI,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     logger.NewStdLogger(log, logger.LevelError),
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info(ctx, "startup", "status", "api router started", "host", api.Addr)

		serverErrors <- api.ListenAndServe()
	}()

	// -------------------------------------------------------------------------
	// Shutdown

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info(ctx, "shutdown", "status", "shutdown started", "signal", sig)
		defer log.Info(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(ctx, cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := api.Shutdown(ctx); err != nil {
			api.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}
