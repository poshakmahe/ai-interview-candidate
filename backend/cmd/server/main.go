package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/katim/secure-doc-vault/internal/config"
	"github.com/katim/secure-doc-vault/internal/database"
	"github.com/katim/secure-doc-vault/internal/handlers"
	"github.com/katim/secure-doc-vault/internal/middleware"
	"github.com/katim/secure-doc-vault/internal/services"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize services
	userService := services.NewUserService(db)
	documentService := services.NewDocumentService(db, cfg.UploadDir)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTSecret)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userService, authMiddleware)
	documentHandler := handlers.NewDocumentHandler(documentService, cfg.MaxFileSize)

	// Setup router
	router := gin.Default()

	// Apply CORS middleware
	router.Use(middleware.CORS(cfg.AllowedOrigins))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Auth routes (public)
	auth := router.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.GET("/me", authMiddleware.Authenticate(), authHandler.GetMe)
	}

	// Document routes (protected)
	documents := router.Group("/documents")
	documents.Use(authMiddleware.Authenticate())
	{
		documents.GET("", documentHandler.ListDocuments)
		documents.POST("", documentHandler.UploadDocument)
		documents.GET("/:id", documentHandler.GetDocument)
		documents.PATCH("/:id", documentHandler.RenameDocument)
		documents.DELETE("/:id", documentHandler.DeleteDocument)
		documents.GET("/:id/download", documentHandler.DownloadDocument)
		documents.POST("/:id/share", documentHandler.ShareDocument)
	}

	// Shared documents route (protected)
	router.GET("/shared", authMiddleware.Authenticate(), documentHandler.ListSharedDocuments)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down gracefully...")
		os.Exit(0)
	}()

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
