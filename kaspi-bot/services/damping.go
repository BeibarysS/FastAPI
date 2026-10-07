package services

import (
    "kaspi-bot/db"
    "kaspi-bot/models"
    "kaspi-bot/scraper"
    "kaspi-bot/kaspi"
    "fmt"
    "log"
    "sync"
    "time"
)

// Per-user damping state
var (
    dampingMutex    sync.RWMutex
    userDampingState = make(map[string]*UserDampingState)
)

type UserDampingState struct {
    IsRunning      bool
    LastRun        time.Time
    DampingResults []DampingResult
}

type DampingResult struct {
    ProductName     string    `json:"product_name"`
    OldPrice        float64   `json:"old_price"`
    NewPrice        float64   `json:"new_price"`
    CompetitorPrice float64   `json:"competitor_price"`
    Success         bool      `json:"success"`
    Error           string    `json:"error,omitempty"`
    UpdatedAt       time.Time `json:"updated_at"`
}

func getUserDampingState(userID string) *UserDampingState {
    dampingMutex.Lock()
    defer dampingMutex.Unlock()
    
    if _, exists := userDampingState[userID]; !exists {
        userDampingState[userID] = &UserDampingState{
            IsRunning:      false,
            DampingResults: []DampingResult{},
        }
    }
    return userDampingState[userID]
}

func StartDamping(userID string) error {
    state := getUserDampingState(userID)
    
    dampingMutex.Lock()
    defer dampingMutex.Unlock()
    
    if state.IsRunning {
        return fmt.Errorf("damping is already running for this user")
    }
    
    go runDampingProcess(userID)
    return nil
}

func runDampingProcess(userID string) {
    state := getUserDampingState(userID)
    
    dampingMutex.Lock()
    state.IsRunning = true
    state.DampingResults = []DampingResult{} // Clear previous results
    dampingMutex.Unlock()
    
    defer func() {
        dampingMutex.Lock()
        state.IsRunning = false
        state.LastRun = time.Now()
        dampingMutex.Unlock()
    }()
    
    log.Printf("🎯 Starting comprehensive price damping process for user: %s", userID)
    
    // Get shop credentials for this user
    shop, err := db.GetLatestShop(userID)
    if err != nil {
        log.Printf("❌ Failed to get shop credentials for user %s: %v", userID, err)
        return
    }
    
    // Get all products for this user
    products, err := db.GetAllProducts(userID)
    if err != nil {
        log.Printf("❌ Failed to get products for user %s: %v", userID, err)
        return
    }
    
    log.Printf("📦 Processing %d products for damping (user: %s)", len(products), userID)
    
    successCount := 0
    
    for i, product := range products {
        log.Printf("🔍 [%d/%d] Processing: %s (user: %s)", i+1, len(products), product.Name, userID)
        
        result := DampingResult{
            ProductName: product.Name,
            OldPrice:    product.Price,
            UpdatedAt:   time.Now(),
        }
        
        // Step 1: Get competitor price
        log.Printf("   📊 Getting competitor prices...")
        competitorPrice, err := scraper.GetLowestCompetitorPrice(product.Name)
        if err != nil {
            log.Printf("   ⚠️ Failed to get competitor price: %v", err)
            result.Error = fmt.Sprintf("Failed to get competitor price: %v", err)
            result.Success = false
            addDampingResult(userID, result)
            continue
        }
        
        result.CompetitorPrice = competitorPrice
        log.Printf("   💰 Competitor lowest price: %.0f KZT", competitorPrice)
        
        // Step 2: Calculate optimal price
        var newPrice float64
        if competitorPrice <= product.MinPrice {
            newPrice = product.MinPrice
            log.Printf("   ⬆️ Competitor price too low, using min price: %.0f KZT", newPrice)
        } else {
            newPrice = competitorPrice - 1
            log.Printf("   🎯 Setting price 1 KZT below competitor: %.0f KZT", newPrice)
        }
        
        // Apply max price constraint
        if newPrice > product.MaxPrice {
            newPrice = product.MaxPrice
            log.Printf("   ⬇️ Adjusted to max price: %.0f KZT", newPrice)
        }
        
        result.NewPrice = newPrice
        
        // Step 3: Check if price change is needed
        if product.Price == newPrice {
            log.Printf("   ⏭️ Price unchanged (%.0f KZT), skipping update", newPrice)
            result.Success = true
            result.Error = "No price change needed"
            addDampingResult(userID, result)
            continue
        }
        
        // Step 4: Check if price change is significant enough
        priceDifference := abs(product.Price - newPrice)
        if priceDifference < 5 { // Less than 5 KZT difference
            log.Printf("   ⏭️ Price difference too small (%.0f KZT), skipping update", priceDifference)
            result.Success = true
            result.Error = "Price difference too small, no update needed"
            addDampingResult(userID, result)
            continue
        }
        
        log.Printf("   🎯 Updating price: %.0f → %.0f KZT (difference: %.0f)", product.Price, newPrice, priceDifference)
        
        // Step 5: Update price on Kaspi website
        log.Printf("   🌐 Updating price on Kaspi website...")
        err = kaspi.UpdateProductPriceOnKaspi(shop.Login, shop.Password, product.Name, newPrice)
        if err != nil {
            log.Printf("   ❌ Failed to update price on Kaspi: %v", err)
            result.Error = fmt.Sprintf("Failed to update on Kaspi: %v", err)
            result.Success = false
            addDampingResult(userID, result)
            continue
        }
        
        // Step 6: Update price in local database (with user_id filtering)
        log.Printf("   💾 Updating price in database...")
        err = db.UpdateProductPrice(product.ID, newPrice, userID)
        if err != nil {
            log.Printf("   ⚠️ Failed to update price in database: %v", err)
            result.Error = fmt.Sprintf("Updated on Kaspi but failed to save in DB: %v", err)
            result.Success = false
            addDampingResult(userID, result)
            continue
        }
        
        result.Success = true
        successCount++
        
        log.Printf("   ✅ Successfully updated %s: %.0f → %.0f KZT", product.Name, product.Price, newPrice)
        
        addDampingResult(userID, result)
        
        // Small delay to avoid overwhelming servers
        time.Sleep(3 * time.Second)
    }
    
    log.Printf("🎉 Damping completed for user %s! Successfully updated %d out of %d products", userID, successCount, len(products))
    
    // Log summary
    state = getUserDampingState(userID)
    if len(state.DampingResults) > 0 {
        log.Printf("📊 Damping Summary for user %s:", userID)
        for _, result := range state.DampingResults {
            status := "✅"
            if !result.Success {
                status = "❌"
            }
            log.Printf("   %s %s: %.0f → %.0f KZT", status, result.ProductName, result.OldPrice, result.NewPrice)
        }
    }
}

func addDampingResult(userID string, result DampingResult) {
    dampingMutex.Lock()
    defer dampingMutex.Unlock()
    
    state := userDampingState[userID]
    if state != nil {
        state.DampingResults = append(state.DampingResults, result)
    }
}

func GetDampingStatus(userID string) models.DampingStatus {
    state := getUserDampingState(userID)
    products, _ := db.GetAllProducts(userID)
    
    dampingMutex.RLock()
    defer dampingMutex.RUnlock()
    
    status := models.DampingStatus{
        IsRunning:     state.IsRunning,
        ProductsCount: len(products),
    }
    
    if !state.LastRun.IsZero() {
        status.LastRun = state.LastRun.Format("2006-01-02 15:04:05")
    }
    
    if state.IsRunning {
        status.Message = fmt.Sprintf("Damping in progress... Processed %d products", len(state.DampingResults))
    } else if len(state.DampingResults) > 0 {
        successCount := 0
        for _, result := range state.DampingResults {
            if result.Success {
                successCount++
            }
        }
        status.Message = fmt.Sprintf("Last run completed: %d/%d products updated successfully", successCount, len(state.DampingResults))
    } else {
        status.Message = "Damping process is idle"
    }
    
    return status
}

func GetDampingResults(userID string) []DampingResult {
    state := getUserDampingState(userID)
    
    dampingMutex.RLock()
    defer dampingMutex.RUnlock()
    
    return state.DampingResults
}

func abs(x float64) float64 {
    if x < 0 {
        return -x
    }
    return x
}
