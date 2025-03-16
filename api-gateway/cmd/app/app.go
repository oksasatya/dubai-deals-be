package main

import (
	"api-gateway/config"
	"api-gateway/handler"
	"api-gateway/routes"
	"api-gateway/webResponse"
	"context"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/time/rate"
	"messaging"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

type App struct {
	Server          *echo.Echo
	Handler         *Handler
	DB              *mongo.Database
	RMQ             *messaging.RabbitMQConnection
	ResponseHandler *webResponse.ResponseHandler
}

// Handler Struct for saving instance of handler
type Handler struct {
	UserHandler  *handler.UserHandler
	AdminHandler *handler.AdminHandler
}

type RegisterRoutes struct {
}

// Initialize sets up environment and app
func (app *App) Initialize() {
	app.LoadEnv()
	app.Server = echo.New()

	// Recover from panic
	defer func() {
		if r := recover(); r != nil {
			logrus.Fatalf("Panic occurred during initialization: %v", r)
		}
	}()

	// Rate Limit
	cfg := config.LoadRateLimitConfig()
	app.Server.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(
		middleware.RateLimiterMemoryStoreConfig{
			Rate:      rate.Limit(cfg.RateLimit),
			Burst:     cfg.RateLimit,
			ExpiresIn: 1 * time.Minute,
		},
	)))

	// Redis
	config.InitRedis()

	// Logger
	config.SetupLogger()

	// Middleware
	app.Server.Use(middleware.Recover())
	app.Server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))
	app.Server.Use(middleware.Gzip())
	app.Server.OPTIONS("/*", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	rmq, err := messaging.NewRabbitMQConnection()
	if err != nil {
		logrus.Fatalf("Failed to initialize RabbitMQ: %v", err)
	}
	app.RMQ = rmq
	logrus.Info("RabbitMQ initialized successfully")

	if app.RMQ == nil {
		logrus.Fatal("Failed to initialize RabbitMQ")
	}
	app.Handler = &Handler{
		UserHandler:  handler.NewUserHandler(cfg, app.RMQ, app.ResponseHandler),
		AdminHandler: handler.NewAdminHandler(cfg, app.RMQ, app.ResponseHandler),
	}
	app.ResponseHandler = webResponse.NewResponseHandler(app.RMQ)
	if app.ResponseHandler == nil {
		logrus.Fatal("ResponseHandler is nil after initialization")
	}

	if app.Handler == nil || app.Handler.UserHandler == nil {
		logrus.Fatal("Failed to initialize handler")
	}

	// Initialize routes
	registerRoutes := routes.RegisterRoutes{
		Echo: app.Server,
		Cfg:  cfg,
		RMQ:  app.RMQ,
		Res:  app.ResponseHandler,
	}
	registerRoutes.RegisterAllRoutes()
}

// LoadEnv function to load environment variables
func (app *App) LoadEnv() {
	cwd, err := os.Getwd()
	if err != nil {
		logrus.Fatal("Error getting current working directory:", err)
	}

	envPath := filepath.Join(cwd, "..", ".env")
	if err := godotenv.Load(envPath); err != nil {
		logrus.Println("Warning: No .env file found, continuing...")
	}

	var devEnvPath string
	if os.Getenv("RUNNING_IN_DOCKER") == "true" {
		devEnvPath = "/app/.env"
	} else if os.Getenv("APP_ENV") == "development" {
		devEnvPath = filepath.Join(cwd, "..", ".env.development")
	} else {
		devEnvPath = filepath.Join(cwd, "..", ".env")
	}

	if err := godotenv.Load(devEnvPath); err != nil {
		logrus.Fatal("Error loading .env file:", err)
	}
	logrus.Println("Successfully loaded .env from:", devEnvPath)
}

// Run starts the server
func (app *App) Run() {
	port := os.Getenv("API_GATEWAY_PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		if err := app.Server.Start(":" + port); err != nil {
			logrus.Fatalf("Server stopped unexpectedly: %v", err)
		}
	}()
	app.handleShutdown()

}

// handleShutdown function to gracefully shutdown server
func (app *App) handleShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logrus.Warn("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Server.Shutdown(ctx); err != nil {
		logrus.Fatalf("Error shutting down server: %v", err)
	}

	if app.RMQ != nil {
		app.RMQ.Close()
	}

	logrus.Info("Server and RabbitMQ connection closed successfully")
}
