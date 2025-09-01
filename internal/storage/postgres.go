package storage

import (
	"context"
	"time"

	"github.com/aldevkode/order-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PG struct { Pool *pgxpool.Pool };

func New(ctx context.Context, dsn string) (*PG, error) {
    pool, err := pgxpool.New(ctx, dsn)
    if err != nil { return nil, err }
    ctx2, cancel := context.WithTimeout(ctx, 5*time.Second); defer cancel()
    if err := pool.Ping(ctx2); err != nil { pool.Close(); return nil, err }
    return &PG{Pool: pool}, nil
}

func (p *PG) Close() { p.Pool.Close() }

func (p *PG) InsertOrder(ctx context.Context, o model.Order) error {
    tx, err := p.Pool.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback(ctx)

    // orders
    _, err = tx.Exec(ctx, `INSERT INTO orders(order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
      ON CONFLICT (order_uid) DO NOTHING`,
      o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerID, o.DeliveryService, o.ShardKey, o.SmID, o.DateCreated, o.OofShard,
    )
    if err != nil { return err }

    // delivery
    _, err = tx.Exec(ctx, `INSERT INTO deliveries(order_uid, name, phone, zip, city, address, region, email)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
      ON CONFLICT (order_uid) DO UPDATE SET name=EXCLUDED.name, phone=EXCLUDED.phone, zip=EXCLUDED.zip, city=EXCLUDED.city, address=EXCLUDED.address, region=EXCLUDED.region, email=EXCLUDED.email`,
      o.OrderUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City, o.Delivery.Address, o.Delivery.Region, o.Delivery.Email,
    )
    if err != nil { return err }

    // payment
    _, err = tx.Exec(ctx, `INSERT INTO payment(order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
      ON CONFLICT (order_uid) DO UPDATE SET transaction=EXCLUDED.transaction, request_id=EXCLUDED.request_id, currency=EXCLUDED.currency, provider=EXCLUDED.provider, amount=EXCLUDED.amount, payment_dt=EXCLUDED.payment_dt, bank=EXCLUDED.bank, delivery_cost=EXCLUDED.delivery_cost, goods_total=EXCLUDED.goods_total, custom_fee=EXCLUDED.custom_fee`,
      o.OrderUID, o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider, o.Payment.Amount, o.Payment.PaymentDT, o.Payment.Bank, o.Payment.DeliveryCost, o.Payment.GoodsTotal, o.Payment.CustomFee,
    )
    if err != nil { return err }

    // items
    _, err = tx.Exec(ctx, `DELETE FROM items WHERE order_uid=$1`, o.OrderUID)
    if err != nil { return err }
    for _, it := range o.Items {
        _, err = tx.Exec(ctx, `INSERT INTO items(order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
          VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
          o.OrderUID, it.ChrtID, it.TrackNumber, it.Price, it.RID, it.Name, it.Sale, it.Size, it.TotalPrice, it.NmID, it.Brand, it.Status,
        )
        if err != nil { return err }
    }

    return tx.Commit(ctx)
}

func (p *PG) GetOrder(ctx context.Context, id string) (model.Order, bool, error) {
    var o model.Order
    err := p.Pool.QueryRow(ctx, `SELECT order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard FROM orders WHERE order_uid=$1`, id).
        Scan(&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature, &o.CustomerID, &o.DeliveryService, &o.ShardKey, &o.SmID, &o.DateCreated, &o.OofShard)
    if err != nil { return model.Order{}, false, err }
    // delivery
    _ = p.Pool.QueryRow(ctx, `SELECT name, phone, zip, city, address, region, email FROM deliveries WHERE order_uid=$1`, id).
        Scan(&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip, &o.Delivery.City, &o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email)
    // payment
    _ = p.Pool.QueryRow(ctx, `SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee FROM payment WHERE order_uid=$1`, id).
        Scan(&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency, &o.Payment.Provider, &o.Payment.Amount, &o.Payment.PaymentDT, &o.Payment.Bank, &o.Payment.DeliveryCost, &o.Payment.GoodsTotal, &o.Payment.CustomFee)
    // items
    rows, err := p.Pool.Query(ctx, `SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status FROM items WHERE order_uid=$1`, id)
    if err == nil {
        defer rows.Close()
        for rows.Next() {
            var it model.Item
            if err := rows.Scan(&it.ChrtID, &it.TrackNumber, &it.Price, &it.RID, &it.Name, &it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status); err == nil {
                o.Items = append(o.Items, it)
            }
        }
    }
    return o, true, nil
}

func (p *PG) Recent(ctx context.Context, n int) ([]model.Order, error) {
    rows, err := p.Pool.Query(ctx, `SELECT order_uid FROM orders ORDER BY date_created DESC LIMIT $1`, n)
    if err != nil { return nil, err }
    defer rows.Close()
    ids := make([]string,0,n)
    for rows.Next() { var id string; _ = rows.Scan(&id); ids = append(ids,id) }
    out := make([]model.Order, 0, len(ids))
    for _, id := range ids {
        if o, ok, err := p.GetOrder(ctx, id); err == nil && ok { out = append(out, o) }
    }
    return out, nil
}