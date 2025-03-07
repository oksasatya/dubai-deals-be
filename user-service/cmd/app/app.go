package main

import (
	"api-gateway/config"
	"context"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"messaging"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"user-service/api"
	"user-service/core/models"
	"user-service/core/repository"
	"user-service/core/service"
	"user-service/database"
	"user-service/eventhandlers/admin"
	"user-service/eventhandlers/user"
	"user-service/utils"
)

// App struct for save instance of app
type App struct {
	DB      *mongo.Database
	Server  *echo.Echo
	Service *Service
	RMQ     *messaging.RabbitMQConnection
}

type Service struct {
	UserService  service.UserService
	AdminService service.AdminService
}

// Initialize prepare environment and setup app
func (app *App) Initialize() {
	app.LoadEnv()

	// Init Server
	app.Server = echo.New()

	// Init Database
	db, err := database.InitMongoDB()
	if err != nil {
		logrus.Fatalf("Error connecting to database: %v", err)
	}
	app.DB = db

	config.SetupLogger()

	// Init RabbitMQ
	rmq, err := messaging.NewRabbitMQConnection()
	if err != nil {
		logrus.Fatalf("Failed to initialize RabbitMQ: %v", err)
	}
	app.RMQ = rmq

	// Init Service
	app.Service = &Service{
		UserService:  service.NewUserService(repository.NewUserRepo(db), rmq, api.NewSendingMessage(rmq)),
		AdminService: service.NewAdminService(repository.NewUserRepo(db), rmq, api.NewSendingMessage(rmq)),
	}
}

// RunConsumer function to run consumer
func (app *App) RunConsumer(wg *sync.WaitGroup) {
	defer wg.Done()
	if app.Service == nil {
		logrus.Fatal("[admin-service] app.Service is nil in RunConsumer!")
	}

	if app.Service.AdminService == nil {
		logrus.Fatal("[admin-service] AdminService is nil in RunConsumer!")
	}

	eventHandlers := map[string]func(models.Event){
		// User
		"UserRegistered": func(event models.Event) {
			user.HandleRegisterEvent(context.Background(), utils.MarshalPayload(event.Payload), event.CorrelationID, app.Service.UserService)
		},

		"UserLogin": func(event models.Event) {
			user.HandleLoginEvent(context.Background(), utils.MarshalPayload(event.Payload), event.CorrelationID, app.Service.UserService)
		},

		"GetProfile": func(event models.Event) {
			user.GetProfileUser(context.Background(), utils.MarshalPayload(event.Payload), event.CorrelationID, app.Service.UserService)
		},
		"UserLogout": func(event models.Event) {
			user.HandleLogoutEvent(context.Background(), utils.MarshalPayload(event.Payload), event.CorrelationID, app.Service.UserService)
		},

		// Admin
		"AdminCreate": func(event models.Event) {
			admin.HandleCreateAdmin(context.Background(), utils.MarshalPayload(event.Payload), event.CorrelationID, app.Service.AdminService)
		},

		"AdminUpdated": func(event models.Event) {
			admin.HandleAdminUpdateEvent(context.Background(), utils.MarshalPayload(event.Payload), event.CorrelationID, app.Service.AdminService)
		},

		"GetAllAdmin": func(event models.Event) {
			admin.GetAllAdminEvent(context.Background(), utils.MarshalPayload(event.Payload), event.CorrelationID, app.Service.AdminService)
		},
	}

	var eventNames []string
	for eventName := range eventHandlers {
		eventNames = append(eventNames, eventName)
	}

	go func() {
		logrus.Infof("[RabbitMQ] Listening for events: %v", eventNames)
		messaging.ConsumeEvent(app.RMQ, "user-service", eventNames, func(event models.Event) {
			if handler, exists := eventHandlers[event.EventType]; exists {
				handler(event)
			} else {
				logrus.Warnf("No handler found for event: %s", event.EventType)
			}
		})
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan
	logrus.Warn("[RabbitMQ] Stopping consumers...")
}

// Run function to run the app
func (app *App) Run() {
	port := os.Getenv("USER_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	var wg sync.WaitGroup
	wg.Add(1)
	// Run Consumer
	go app.RunConsumer(&wg)

	go func() {
		if err := app.Server.Start(":" + port); err != nil {
			logrus.Info("Shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	logrus.Info("Server shutdown")
	wg.Wait()
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
		logrus.Fatal("Error loading .env.development file:", err)
	}
	logrus.Println("Successfully loaded .env.development from:", devEnvPath)
}
