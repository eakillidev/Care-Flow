package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/auth"
	"github.com/eakillidev/Care-Flow/backend/internal/caregivers"
	"github.com/eakillidev/Care-Flow/backend/internal/config"
	"github.com/eakillidev/Care-Flow/backend/internal/database"
	"github.com/eakillidev/Care-Flow/backend/internal/patients"
	"github.com/eakillidev/Care-Flow/backend/internal/users"
	"github.com/eakillidev/Care-Flow/backend/internal/visits"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.Load()

	dbContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.Connect(dbContext, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()
	log.Print("database connection established")

	tokens, err := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTExpiry)
	if err != nil {
		log.Fatalf("configure JWT authentication: %v", err)
	}
	userRepository := users.NewPostgresRepository(pool)
	patientRepository := patients.NewPostgresRepository(pool)
	visitRepository := visits.NewPostgresRepository(pool)
	authService := auth.NewService(userRepository, tokens)
	authHandler := auth.NewHandler(authService, userRepository)
	patientHandler := patients.NewHandler(patients.NewService(patientRepository))
	caregiverHandler := caregivers.NewHandler(userRepository)
	visitHandler := visits.NewHandler(visits.NewService(visitRepository, patientRepository, userRepository))

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           newRouter(pool, authHandler, patientHandler, caregiverHandler, visitHandler, tokens),
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownSignal.Done()

		shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("server shutdown failed: %v", err)
		}
	}()

	log.Printf("careflow-api listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

type databasePinger interface {
	Ping(context.Context) error
}

func newRouter(
	database databasePinger,
	authHandler *auth.Handler,
	patientHandler *patients.Handler,
	caregiverHandler *caregivers.Handler,
	visitHandler *visits.Handler,
	tokens *auth.TokenManager,
) http.Handler {
	router := chi.NewRouter()
	router.Get("/health", healthHandler(database))
	router.Post("/api/auth/login", authHandler.Login)
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Authenticate(tokens))
		protected.Get("/api/me", authHandler.Me)
		protected.With(auth.RequireRole(users.RoleCoordinator)).Get("/api/coordinator/ping", auth.CoordinatorPing)
		protected.With(auth.RequireRole(users.RoleCaregiver)).Get("/api/caregiver/ping", auth.CaregiverPing)

		if patientHandler != nil && caregiverHandler != nil && visitHandler != nil {
			protected.Group(func(coordinator chi.Router) {
				coordinator.Use(auth.RequireRole(users.RoleCoordinator))
				coordinator.Post("/api/patients", patientHandler.Create)
				coordinator.Get("/api/patients", patientHandler.List)
				coordinator.Get("/api/patients/{id}", patientHandler.Get)
				coordinator.Put("/api/patients/{id}", patientHandler.Update)
				coordinator.Get("/api/caregivers", caregiverHandler.List)
				coordinator.Post("/api/visits", visitHandler.Create)
				coordinator.Get("/api/visits", visitHandler.List)
				coordinator.Get("/api/visits/{id}", visitHandler.Get)
				coordinator.Patch("/api/visits/{id}", visitHandler.UpdateSchedule)
				coordinator.Post("/api/visits/{id}/cancel", visitHandler.Cancel)
			})

			protected.Group(func(caregiver chi.Router) {
				caregiver.Use(auth.RequireRole(users.RoleCaregiver))
				caregiver.Get("/api/caregiver/visits", visitHandler.ListForCaregiver)
				caregiver.Get("/api/caregiver/visits/{id}", visitHandler.GetForCaregiver)
			})
		}
	})
	return router
}

func healthHandler(database databasePinger) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
		defer cancel()

		response := map[string]string{
			"status":   "ok",
			"service":  "careflow-api",
			"database": "ok",
		}
		statusCode := http.StatusOK
		if err := database.Ping(ctx); err != nil {
			response["status"] = "unhealthy"
			response["database"] = "unavailable"
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("write health response: %v", err)
		}
	}
}
