package api

import (
	"github.com/ctrlsam/rigour/internal/redis"
	"github.com/ctrlsam/rigour/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Router provides the Chi router configuration for the API.
type Router struct {
	router    *chi.Mux
	handler   *Handler
	dashboard *HealthDashboard
}

// NewRouter creates a new API router.
func NewRouter(repository storage.HostRepository, redisClient *redis.Client) *Router {
	r := chi.NewRouter()

	// Add CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-API-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Add other middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	handler := NewHandler(repository)
	dashboard := NewHealthDashboard(redisClient)

	// Register routes
	r.Get("/health", handler.HealthHandler)

	r.Route("/api", func(r chi.Router) {
		r.Get("/hosts/search", handler.SearchHandler)
		r.Get("/hosts/{ip}", handler.GetHostHandler)
		r.Get("/facets", handler.FacetsHandler)
		
		// Health dashboard routes
		r.Route("/dashboard", func(r chi.Router) {
			r.Get("/stats", dashboard.GetScanStats)
			r.Get("/schedules", dashboard.GetPortSchedules)
			r.Get("/asn-rates", dashboard.GetASNRates)
			r.Get("/streams", dashboard.GetStreamHealth)
			r.Get("/metrics", dashboard.GetSystemMetrics)
		})
	})

	return &Router{
		router:    r,
		handler:   handler,
		dashboard: dashboard,
	}
}

// Handler returns the underlying Chi router for use with http.ListenAndServe.
func (r *Router) Handler() *chi.Mux {
	return r.router
}
