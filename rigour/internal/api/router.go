package api

import (
	"github.com/ctrlsam/rigour/internal/blocklist"
	"github.com/ctrlsam/rigour/internal/redis"
	"github.com/ctrlsam/rigour/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Router provides the Chi router configuration for the API.
type Router struct {
	router       *chi.Mux
	handler      *Handler
	dashboard    *HealthDashboard
	optOut       *OptOutHandler
	scanInfoCfg  ScanInfoConfig
}

// NewRouter creates a new API router.
func NewRouter(repository storage.HostRepository, redisClient *redis.Client) *Router {
	return NewRouterWithAnalytics(repository, redisClient, nil, nil)
}

// NewRouterWithAnalytics creates a new API router with analytics support.
func NewRouterWithAnalytics(repository storage.HostRepository, redisClient *redis.Client, analytics *AnalyticsHandler, bl *blocklist.Blocklist) *Router {
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
	
	// Initialize opt-out handler if blocklist provided
	var optOutHandler *OptOutHandler
	if bl != nil {
		optOutHandler = NewOptOutHandler(bl, redisClient)
	}

	// Scaninfo configuration
	scanInfoCfg := ScanInfoConfig{
		BaseURL:      "https://rigour.example.com", // TODO: Make configurable via env
		ContactEmail: "security@rigour.example.com",
		GitHubURL:    "https://github.com/coopergrace925/rigour",
		WebsiteURL:   "https://rigour.example.com",
	}

	// Register public routes (before /api)
	r.Get("/scaninfo", HandleScanInfo(scanInfoCfg))
	r.Get("/.well-known/security.txt", HandleSecurityTxt(scanInfoCfg))
	r.Get("/health", handler.HealthHandler)

	r.Route("/api", func(r chi.Router) {
		r.Get("/hosts/search", handler.SearchHandler)
		r.Get("/hosts/{ip}", handler.GetHostHandler)
		r.Get("/facets", handler.FacetsHandler)
		
		// Opt-out routes (public, no auth required)
		if optOutHandler != nil {
			r.Post("/opt-out", optOutHandler.HandleOptOut)
			r.Get("/opt-out", optOutHandler.HandleOptOutList)
		}
		
		// Health dashboard routes
		r.Route("/dashboard", func(r chi.Router) {
			r.Get("/stats", dashboard.GetScanStats)
			r.Get("/schedules", dashboard.GetPortSchedules)
			r.Get("/asn-rates", dashboard.GetASNRates)
			r.Get("/streams", dashboard.GetStreamHealth)
			r.Get("/metrics", dashboard.GetSystemMetrics)
		})
		
		// Analytics routes
		if analytics != nil {
			r.Route("/analytics", func(r chi.Router) {
				r.Get("/overview", analytics.GetOverview)
				r.Get("/services", analytics.GetServiceAnalytics)
				r.Get("/cves", analytics.GetCVEAnalytics)
				r.Get("/asns", analytics.GetASNAnalytics)
			})
		}
	})

	return &Router{
		router:      r,
		handler:     handler,
		dashboard:   dashboard,
		optOut:      optOutHandler,
		scanInfoCfg: scanInfoCfg,
	}
}

// Handler returns the underlying Chi router for use with http.ListenAndServe.
func (r *Router) Handler() *chi.Mux {
	return r.router
}
