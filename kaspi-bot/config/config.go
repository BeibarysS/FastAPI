package config

import (
    "log"
    "os"

    "github.com/joho/godotenv"
)

var (
    SupabaseURL string
    SupabaseKey string
    JWTSecret   string
)

func LoadEnv() {
    err := godotenv.Load()
    if err != nil {
        log.Println("Warning: .env file not found, using environment variables")
    }

    SupabaseURL = os.Getenv("SUPABASE_URL")
    SupabaseKey = os.Getenv("SUPABASE_KEY")
    JWTSecret = os.Getenv("JWT_SECRET")

    if SupabaseURL == "" || SupabaseKey == "" {
        log.Fatal("❌ SUPABASE_URL and SUPABASE_KEY must be set")
    }
}
