package main

import (
    "context"
    "log"
    "net/http"
    "os/signal"
    "syscall"

    "github.com/aldevkode/order-service/internal/cache"
    "github.com/aldevkode/order-service/internal/config"
    apiserver "github.com/aldevkode/order-service/internal/http"
    "github.com/aldevkode/order-service/internal/model"
    "github.com/aldevkode/order-service/internal/kafka"
    "github.com/aldevkode/order-service/internal/storage"
)

func main(){
    cfg := config.Load()

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    db, err := storage.New(ctx, cfg.PostgresDSN)
    if err != nil { log.Fatalf("db: %v", err) }
    defer db.Close()

    c := cache.New()
    if recent, err := db.Recent(ctx, cfg.CacheWarmN); err == nil { c.BulkSet(recent) }

    handler := func(ctx context.Context, o model.Order) error {
    if err := db.InsertOrder(ctx, o); err != nil { 
        return err 
    }
    c.Set(o)
    log.Printf("order stored: %s items=%d", o.OrderUID, len(o.Items))
    return nil
}

    go func(){
        cons := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, "order-demo", handler)
        if err := cons.Run(ctx); err != nil { log.Printf("kafka stopped: %v", err) }
    }()

    srv := apiserver.New(c)
    go func(){
        log.Printf("HTTP on %s", cfg.HTTPAddr)
        if err := http.ListenAndServe(cfg.HTTPAddr, srv.Handler()); err != nil { log.Printf("http: %v", err) }
    }()

    <-ctx.Done()
    log.Println("shutdown")
}