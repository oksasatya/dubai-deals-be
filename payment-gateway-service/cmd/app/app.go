package main

import (
	configApiGateway "api-gateway/config"
	"context"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"mail-service/config"
	"mail-service/database"
	"messaging"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

type App struct {
	Server *echo.Echo
	RMQ    *messaging.RabbitMQConnection
}

var db *gorm.DB

func (app *App) Initialize() {
	app.LoadEnv()
	app.Server = echo.New()

	// Recover from panic
	defer func() {
		if r := recover(); r != nil {
			logrus.Fatalf("Panic occurred during initialization: %v", r)
		}
	}()

	// RabbitMQ
	rmq, err := messaging.NewRabbitMQConnection()
	if err != nil {
		logrus.Fatalf("Failed to initialize RabbitMQ: %v", err)
	}
	app.RMQ = rmq
	logrus.Info("RabbitMQ initialized successfully")
	if app.RMQ == nil {
		logrus.Fatal("Failed to initialize RabbitMQ")
	}

	// logger
	configApiGateway.SetupLogger()

	// midtrans
	midtransClient := config.InitMidtrans()
	if midtransClient == nil {
		logrus.Fatal("Failed to initialize Midtrans")
	}

	// Init DB
	db, err = database.InitDB()
	if err != nil {
		logrus.Fatalf("Error connecting to database: %v", err)
	}

}

// Run starts the server
func (app *App) Run() {
	port := os.Getenv("PAYMENT_GATEWAY_SERVICE_PORT")
	if port == "" {
		port = "8083"
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
