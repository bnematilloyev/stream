DROP INDEX IF EXISTS idx_channels_marketplace_shop_id;
DROP INDEX IF EXISTS idx_channels_marketplace_seller_id;
ALTER TABLE channels
    DROP COLUMN IF EXISTS marketplace_shop_id,
    DROP COLUMN IF EXISTS marketplace_seller_id;
