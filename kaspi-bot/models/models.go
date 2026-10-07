package models

type User struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type Shop struct {
    // ID       int    `json:"id,omitempty"`
    Login    string `json:"login"`
    Password string `json:"password"`
    UserID   string `json:"user_id"`
}

type Product struct {
    ID       string    `json:"id,omitempty"`
    UserID   string  `json:"user_id"`
    Name     string  `json:"name"`
    Price    float64 `json:"price"`
    MinPrice float64 `json:"min_price"`
    MaxPrice float64 `json:"max_price"`
    Image    string  `json:"image"`
    URL      string  `json:"url,omitempty"`
}

type CompetitorPrice struct {
    SellerName string  `json:"seller_name"`
    Price      float64 `json:"price"`
}

type DampingStatus struct {
    IsRunning     bool   `json:"is_running"`
    LastRun       string `json:"last_run"`
    ProductsCount int    `json:"products_count"`
    Message       string `json:"message"`
}