// db/db.go
package db

import (
    "kaspi-bot/config"
    "kaspi-bot/models"
    "fmt"
    "log"

    "github.com/nedpals/supabase-go"
)

var Client *supabase.Client

func Init() {
    Client = supabase.CreateClient(config.SupabaseURL, config.SupabaseKey)
    log.Println("✅ Database initialized")
}

// Shop operations with proper user_id handling
func SaveShop(shop models.Shop, userID string) error {
    // Ensure the shop has the correct user_id
    shop.UserID = userID
    
    var result interface{}
    err := Client.DB.From("kaspi_shops").Insert(shop).Execute(&result)
    return err
}

func GetLatestShop(userID string) (models.Shop, error) {
    var shops []models.Shop
    
    err := Client.DB.
    From("kaspi_shops").
    Select("*").
    Eq("user_id", userID).
    Execute(&shops)

    if err != nil {
        return models.Shop{}, err
    }
    
    if len(shops) == 0 {
        return models.Shop{}, fmt.Errorf("no shop credentials found for this user")
    }
    
    return shops[0], nil
}

// Product operations with proper user_id handling
func SaveProducts(products []models.Product, userID string) error {
    if len(products) == 0 {
        return nil
    }
    
    // CRITICAL: Only delete THIS user's products, not all products!
    // This prevents deleting other users' data
    var deleteResult interface{}
    err := Client.DB.From("products").
        Delete().
        Eq("user_id", userID). // Only delete current user's products
        Execute(&deleteResult)
    
    if err != nil {
        log.Printf("Warning: failed to clear existing products for user %s: %v", userID, err)
    }
    
    // Set user_id for all products and insert them
    for _, product := range products {
        product.UserID = userID // Ensure each product has the correct user_id
        
        var result interface{}
        err := Client.DB.From("products").Insert(product).Execute(&result)
        if err != nil {
            log.Printf("Failed to save product %s for user %s: %v", product.Name, userID, err)
            continue
        }
    }
    
    return nil
}

func GetAllProducts(userID string) ([]models.Product, error) {
    var products []models.Product
    err := Client.DB.From("products").
        Select("*").
        Eq("user_id", userID). // Only get this user's products
        Execute(&products)
    return products, err
}

func UpdateProductPrice(id string, price float64, userID string) error {
    update := map[string]interface{}{
        "price": price,
    }
    var result interface{}
    
    // CRITICAL: Include user_id in the WHERE clause to prevent 
    // users from updating other users' products
    err := Client.DB.From("products").
        Update(update).
        Eq("id", id).
        Eq("user_id", userID). // Security: only update if it belongs to this user
        Execute(&result)
    return err
}

// Helper functions for backward compatibility (remove these after migration)
// These are the OLD functions without user isolation - DANGEROUS!
func SaveShop2(shop models.Shop) error {
    log.Println("⚠️ WARNING: Using deprecated SaveShop2 without user isolation!")
    var result interface{}
    return Client.DB.From("kaspi_shops").Insert(shop).Execute(&result)
}

func GetLatestShop2() (models.Shop, error) {
    log.Println("⚠️ WARNING: Using deprecated GetLatestShop2 without user isolation!")
    var shops []models.Shop
    
    err := Client.DB.
        From("kaspi_shops").
        Select("*").
        Limit(1).
        Execute(&shops)
    
    if err != nil {
        return models.Shop{}, err
    }
    
    if len(shops) == 0 {
        return models.Shop{}, fmt.Errorf("no shop credentials found")
    }
    
    return shops[0], nil
}

func SaveProducts2(products []models.Product) error {
    log.Println("⚠️ WARNING: Using deprecated SaveProducts2 - THIS DELETES ALL USERS' PRODUCTS!")
    if len(products) == 0 {
        return nil
    }
    
    // THIS IS DANGEROUS - it deletes ALL products from ALL users!
    var deleteResult interface{}
    err := Client.DB.From("products").Delete().Neq("price", "0").Execute(&deleteResult)
    
    if err != nil {
        log.Printf("Warning: failed to clear existing products: %v", err)
    }
    
    // Insert new products
    for _, product := range products {
        var result interface{}
        err := Client.DB.From("products").Insert(product).Execute(&result)
        if err != nil {
            log.Printf("Failed to save product %s: %v", product.Name, err)
            continue
        }
    }
    
    return nil
}

func GetAllProducts2() ([]models.Product, error) {
    log.Println("⚠️ WARNING: Using deprecated GetAllProducts2 without user isolation!")
    var products []models.Product
    err := Client.DB.From("products").Select("*").Execute(&products)
    return products, err
}

func UpdateProductPrice2(id string, price float64) error {
    log.Println("⚠️ WARNING: Using deprecated UpdateProductPrice2 without user isolation!")
    update := map[string]interface{}{
        "price": price,
    }
    var result interface{}
    err := Client.DB.From("products").Update(update).Eq("id", id).Execute(&result)
    return err
}