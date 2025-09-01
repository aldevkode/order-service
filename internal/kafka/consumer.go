package kafka

import (
    "context"
    "encoding/json"
    "log"
    "strings"
    "time"

    kgo "github.com/segmentio/kafka-go"
    "github.com/aldevkode/order-service/internal/model"
)

type Handler func(context.Context, model.Order) error

type Consumer struct {
    reader *kgo.Reader
    handle Handler
}

func NewConsumer(brokers, topic, group string, h Handler) *Consumer {
    r := kgo.NewReader(kgo.ReaderConfig{
        Brokers: strings.Split(brokers, ","),
        Topic:   topic,
        GroupID: group,
        StartOffset: kgo.LastOffset,
        CommitInterval: time.Second,
    })
    return &Consumer{reader: r, handle: h}
}

func (c *Consumer) Run(ctx context.Context) error {
    defer c.reader.Close()
    for {
        m, err := c.reader.FetchMessage(ctx)
        if err != nil { return err }
        var ord model.Order
        if err := json.Unmarshal(m.Value, &ord); err != nil {
            log.Printf("invalid message: %v", err)
            if err := c.reader.CommitMessages(ctx, m); err != nil { log.Printf("commit error: %v", err) }
            continue
        }
        if ord.OrderUID == "" || len(ord.Items) == 0 {
            log.Printf("invalid order: missing key fields")
            if err := c.reader.CommitMessages(ctx, m); err != nil { log.Printf("commit error: %v", err) }
            continue
        }
        if err := c.handle(ctx, ord); err != nil {
            log.Printf("handle error: %v — will retry", err)
            time.Sleep(time.Second)
            continue
        }
        if err := c.reader.CommitMessages(ctx, m); err != nil { log.Printf("commit error: %v", err) }
    }
}