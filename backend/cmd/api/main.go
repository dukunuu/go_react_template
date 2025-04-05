package main

import (
	"context"
	"errors"
	"go-react-template/backend/internal/auth"
	"go-react-template/backend/internal/db"
	"go-react-template/backend/internal/handlers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	initCtx := context.Background()

	dbService, err := db.NewDatabase(initCtx)
	if err != nil {
		log.Fatalf("FATAL: Could not initialize database: %v", err)
	}
	defer dbService.Close()

	authService, err := auth.NewAuthService(initCtx)
	if err != nil {
		log.Fatalf("FATAL: Could not initialize Firebase Auth service: %v", err)
	}

	// --- Instantiate Handlers --- // <-- ADD THIS
	courseHandler := handlers.NewCourseHandler(dbService)
	taskHandler := handlers.NewTaskHandler(dbService)
	// userHandler := handlers.NewUserHandler(dbService) // If you create user handlers

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := dbService.Pool.Ping(pingCtx); err != nil {
			http.Error(w, "Database connection failed", http.StatusServiceUnavailable)
			log.Printf("WARN: Health check failed database ping: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Route("/api/v1", func(r chi.Router) {

		r.Route("/users", func(r chi.Router) {
			// Define public user routes if needed
			// e.g., r.Post("/sync", userHandler.SyncUser) // Requires auth
		})

		r.Group(func(r chi.Router) {
			r.Use(authService.Middleware())

			r.Route("/courses", func(r chi.Router) {
				r.Post("/", courseHandler.CreateCourse)
				r.Get("/", courseHandler.ListCourses) // Uses query with earned points
				r.Get("/{courseID}", courseHandler.GetCourse)
				r.Put("/{courseID}", courseHandler.UpdateCourse)
				r.Delete("/{courseID}", courseHandler.DeleteCourse)

				r.Route("/{courseID}/tasks", func(r chi.Router) {
					r.Post("/", taskHandler.CreateTask)
					r.Get("/", taskHandler.ListTasks)
				})
			})

			r.Route("/tasks", func(r chi.Router) {
				r.Get("/{taskID}", taskHandler.GetTask)
				r.Put("/{taskID}", taskHandler.UpdateTask) // Full update
				r.Patch("/{taskID}/status", taskHandler.UpdateTaskStatusOnly) // Partial status update
				r.Delete("/{taskID}", taskHandler.DeleteTask)
			})
		})

	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("INFO: Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("FATAL: Could not start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("INFO: Server shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("WARN: Server shutdown failed: %v", err)
	} else {
		log.Println("INFO: Server gracefully stopped.")
	}
}


