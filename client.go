package main

import (
	"context"
	"flag"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/huobirdcenter/huobi_golang/logging/applogger"
	"github.com/morgine/pkg/config"
	"go.uber.org/zap/zapcore"
	"huobi/routes"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 加载配置文件
	configFile := flag.String("c", "config.toml", "配置文件")
	addr := flag.String("a", ":8886", "监听地址")
	flag.Parse()

	engine := gin.New()

	engine.Use(gin.Logger())

	engine.Use(gin.Recovery())

	engine.Use(cors.New(cors.Config{
		// Set cors and db middleware
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 初始化配置服务
	var configs, err = config.UnmarshalFile(*configFile)
	if err != nil {
		panic(err)
	}
	clients := routes.RegisterRoutes(engine, configs)

	var closeFuncs []func()

	for _, client := range clients {
		closeFuncs = append(closeFuncs, client.Subscribe())
	}
	defer func() {
		for _, closeFunc := range closeFuncs {
			closeFunc()
		}
	}()
	serveHttp(*addr, engine)
}

func serveHttp(addr string, engine *gin.Engine) {
	// 开启服务
	srv := &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  180 * time.Second,
		WriteTimeout: 180 * time.Second,
	}

	applogger.Info("listen and serve http://localhost%s/queue", addr)
	applogger.SetLevel(zapcore.InfoLevel)

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			applogger.Error("服务器已停止: %s", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	applogger.Info("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		applogger.Info("Server forced to shutdown:", err)
	}
	applogger.Info("Done!")
}
