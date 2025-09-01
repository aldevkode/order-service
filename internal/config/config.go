package config

import (
    "os"
)

type Config struct {
    HTTPAddr     string
    PostgresDSN  string 
    KafkaBrokers string
    KafkaTopic   string
    CacheWarmN   int 
}

func getenv(key string, def string) string {
    if v := os.Getenv(key); v != "" { return v }
    return def
}

func Load() Config {
    n := 100
    return Config{
        HTTPAddr:     getenv("HTTP_ADDR", ":8081"),
        PostgresDSN:  getenv("POSTGRES_DSN", "postgres://order:order@localhost:5432/orderdb?sslmode=disable"),
        KafkaBrokers: getenv("KAFKA_BROKERS", "localhost:9092"),
        KafkaTopic:   getenv("KAFKA_TOPIC", "orders"),
        CacheWarmN:   n,
    }
}