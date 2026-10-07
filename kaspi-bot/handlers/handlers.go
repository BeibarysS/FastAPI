package handlers

import (
    "kaspi-bot/db"
    "kaspi-bot/middleware"
    "kaspi-bot/models"
    "kaspi-bot/scraper"
    "kaspi-bot/services"
    "encoding/json"
    "log"
    "net/http"

    "github.com/nedpals/supabase-go"
)

// Response helpers
func sendJSON(w http.ResponseWriter, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, message string, code int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func sendSuccess(w http.ResponseWriter, message string) {
    sendJSON(w, map[string]string{"status": "success", "message": message})
}

// Auth handlers
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
    var user models.User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        sendError(w, "Invalid input", http.StatusBadRequest)
        return
    }

    userResponse, err := db.Client.Auth.SignUp(r.Context(), supabase.UserCredentials{
        Email:    user.Email,
        Password: user.Password,
    })

    if err != nil {
        sendError(w, "Registration failed: "+err.Error(), http.StatusBadRequest)
        return
    }

    sendJSON(w, userResponse)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
    var user models.User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        sendError(w, "Invalid input", http.StatusBadRequest)
        return
    }

    userSession, err := db.Client.Auth.SignIn(r.Context(), supabase.UserCredentials{
        Email:    user.Email,
        Password: user.Password,
    })

    if err != nil {
        sendError(w, "Login failed: "+err.Error(), http.StatusUnauthorized)
        return
    }

    // Generate JWT token
    jwtToken, err := middleware.GenerateJWT(userSession.User.ID, userSession.User.Email)
    if err != nil {
        sendError(w, "Failed to generate token: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Return both Supabase session and JWT token
    response := map[string]interface{}{
        "user":         userSession.User,
        "access_token": userSession.AccessToken,
        "jwt_token":    jwtToken, // Your custom JWT for API authentication
    }

    sendJSON(w, response)
}

// Shop handlers (protected routes)
func SaveShopHandler(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.GetUserIDFromContext(r)
    if !ok {
        sendError(w, "User not authenticated", http.StatusUnauthorized)
        return
    }

    var shop models.Shop
    if err := json.NewDecoder(r.Body).Decode(&shop); err != nil {
        sendError(w, "Invalid input", http.StatusBadRequest)
        return
    }

    if err := db.SaveShop(shop, userID); err != nil {
        sendError(w, "Failed to save shop: "+err.Error(), http.StatusInternalServerError)
        return
    }

    sendSuccess(w, "Shop credentials saved successfully")
}

// Product handlers (protected routes)
func ScrapeProductsHandler(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.GetUserIDFromContext(r)
    if !ok {
        sendError(w, "User not authenticated", http.StatusUnauthorized)
        return
    }

    shop, err := db.GetLatestShop(userID)
    if err != nil {
        sendError(w, "Shop credentials not found", http.StatusBadRequest)
        return
    }

    log.Println("🚀 Starting product scraping for user:", userID)
    products, err := scraper.ScrapeKaspiProducts(shop.Login, shop.Password)
    if err != nil {
        sendError(w, "Failed to scrape products: "+err.Error(), http.StatusInternalServerError)
        return
    }

    if err := db.SaveProducts(products, userID); err != nil {
        sendError(w, "Failed to save products: "+err.Error(), http.StatusInternalServerError)
        return
    }

    log.Printf("✅ Successfully scraped and saved %d products for user %s", len(products), userID)
    sendJSON(w, map[string]interface{}{
        "status":   "success",
        "message":  "Products scraped successfully",
        "count":    len(products),
        "products": products,
    })
}

func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.GetUserIDFromContext(r)
    if !ok {
        sendError(w, "User not authenticated", http.StatusUnauthorized)
        return
    }

    products, err := db.GetAllProducts(userID)
    if err != nil {
        sendError(w, "Failed to get products: "+err.Error(), http.StatusInternalServerError)
        return
    }

    sendJSON(w, products)
}

// Price handlers
type CompetitorRequest struct {
    ProductName string `json:"product_name"`
    ProductURL  string `json:"product_url"`
}

func GetCompetitorPricesHandler(w http.ResponseWriter, r *http.Request) {
    // This endpoint might not need user authentication if it's just getting market data
    var req CompetitorRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendError(w, "Invalid input", http.StatusBadRequest)
        return
    }

    if req.ProductName == "" {
        sendError(w, "Product name is required", http.StatusBadRequest)
        return
    }

    price, err := scraper.GetLowestCompetitorPrice(req.ProductName)
    if err != nil {
        sendError(w, "Failed to get competitor prices: "+err.Error(), http.StatusInternalServerError)
        return
    }

    sendJSON(w, map[string]interface{}{
        "product_name":     req.ProductName,
        "lowest_price":     price,
        "competitor_count": 1,
    })
}

// Damping handlers (protected routes)
func StartDampingHandler(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.GetUserIDFromContext(r)
    if !ok {
        sendError(w, "User not authenticated", http.StatusUnauthorized)
        return
    }

    if err := services.StartDamping(userID); err != nil {
        sendError(w, err.Error(), http.StatusConflict)
        return
    }

    sendSuccess(w, "Damping process started in background")
}

func GetDampingStatusHandler(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.GetUserIDFromContext(r)
    if !ok {
        sendError(w, "User not authenticated", http.StatusUnauthorized)
        return
    }

    status := services.GetDampingStatus(userID)
    sendJSON(w, status)
}