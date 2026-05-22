package main

import (
	"log/slog"
	"os"

	"github.com/LimeAsnet/SubAggregator/internal/config"
	"github.com/LimeAsnet/SubAggregator/internal/database"
	"github.com/LimeAsnet/SubAggregator/internal/handlers"
	"github.com/LimeAsnet/SubAggregator/internal/middleware/logger"
	"github.com/LimeAsnet/SubAggregator/internal/repository"
	"github.com/LimeAsnet/SubAggregator/internal/service"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/LimeAsnet/SubAggregator/docs"
)

// @title           SubAggregator API
// @version         1.0
// @description     API агрегатора подписок.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8082
// @BasePath  /api/v1

// @schemes   http
func main() {
	cfg := config.InitConfig()
	slogLog := logger.New(cfg.Env)

	pool, err := database.New(cfg.Database)
	if err != nil {
		slogLog.Error("database connection failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	subRepo := repository.New(pool)
	subSvc := service.NewSubscriptionService(subRepo)
	subHandler := handlers.NewSubscriptionHandler(subSvc, slogLog)

	router := gin.New()
	router.Use(logger.SlogMiddleware(slogLog))
	router.Use(gin.Recovery())

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := router.Group("/api/v1")
	subHandler.Register(api)

	if err := router.Run(cfg.HttpServer.Host); err != nil {
		slogLog.Error("server stopped", slog.String("error", err.Error()))
	}
}
