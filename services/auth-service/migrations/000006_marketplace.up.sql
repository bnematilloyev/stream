ALTER TABLE channels
    ADD COLUMN marketplace_seller_id BIGINT,
    ADD COLUMN marketplace_shop_id   BIGINT;

CREATE UNIQUE INDEX idx_channels_marketplace_seller_id
    ON channels(marketplace_seller_id)
    WHERE marketplace_seller_id IS NOT NULL;

CREATE INDEX idx_channels_marketplace_shop_id
    ON channels(marketplace_shop_id)
    WHERE marketplace_shop_id IS NOT NULL;
