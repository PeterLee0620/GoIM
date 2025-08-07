package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/DavidLee0620/GoIM/chat/app/sdk/mux"
	"github.com/DavidLee0620/GoIM/chat/business/chatbus"
	"github.com/DavidLee0620/GoIM/chat/business/chatbus/usermem"
	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/DavidLee0620/GoIM/chat/foundation/web"
	"github.com/ardanlabs/conf/v3"
	"github.com/google/uuid"
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
			Host       string `conf:"default:demo.nats.io"`
			Subject    string `conf:"default:lee-cap"`
			IDFilePath string `conf:"default:chat/zarf/cap"`
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
	// CapID

	fileName := filepath.Join(cfg.NATS.IDFilePath, "cap.id")
	if _, err := os.Stat(fileName); err != nil {

		os.MkdirAll(cfg.NATS.IDFilePath, os.ModePerm)
		f, err := os.Create(fileName)
		if err != nil {
			return fmt.Errorf("id file Create: %w", err)
		}
		if _, err = f.WriteString(uuid.NewString()); err != nil {
			return fmt.Errorf("id file write: %w", err)
		}
		f.Close()
	}
	f, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("id file open: %w", err)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("id file read: %w", err)
	}
	capID, err := uuid.Parse(string(b))
	if err != nil {
		return fmt.Errorf("id file parse: %w", err)
	}
	log.Info(ctx, "startup", "status", "getting cap", "capID", capID)
	// -------------------------------------------------------------------------
	// NATS Connection
	nc, err := nats.Connect(cfg.NATS.Host)
	if err != nil {
		return fmt.Errorf("nats create: %w", err)
	}
	defer nc.Close()
	chat, err := chatbus.New(log, nc, cfg.NATS.Subject, usermem.New(log), capID)
	if err != nil {
		return fmt.Errorf("chat: %w", err)
	}
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
