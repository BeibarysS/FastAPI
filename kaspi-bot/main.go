package main

import (
    "fmt"
    "log"
    "net/http"

    "kaspi-bot/config"
    "kaspi-bot/db"
    "kaspi-bot/handlers"
    "kaspi-bot/middleware"

    "github.com/gorilla/mux"
)

func main() {
    // Load env vars
    config.LoadEnv()

    // Init DB
    db.Init()

    // Router
    r := mux.NewRouter()

    // CORS middleware
    r.Use(corsMiddleware)

    // Public routes (no authentication required)
    r.HandleFunc("/register", handlers.RegisterHandler).Methods("POST")
    r.HandleFunc("/login", handlers.LoginHandler).Methods("POST")
    r.HandleFunc("/competitor-prices", handlers.GetCompetitorPricesHandler).Methods("POST")

    // Protected routes (require JWT authentication)
    protected := r.PathPrefix("/api").Subrouter()
    protected.Use(middleware.JWTMiddleware)
    
    protected.HandleFunc("/shop", handlers.SaveShopHandler).Methods("POST")
    protected.HandleFunc("/products/scrape", handlers.ScrapeProductsHandler).Methods("POST")
    protected.HandleFunc("/products", handlers.GetProductsHandler).Methods("GET")
    protected.HandleFunc("/damping/start", handlers.StartDampingHandler).Methods("POST")
    protected.HandleFunc("/damping/status", handlers.GetDampingStatusHandler).Methods("GET")

    // DEBUG: print all registered routes
    r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
        path, _ := route.GetPathTemplate()
        methods, _ := route.GetMethods()
        fmt.Println("📌 Route:", methods, path)
        return nil
    })

    fmt.Println("🚀 Kaspi Bot Server started at :8080")
    log.Fatal(http.ListenAndServe(":8080", r))
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}