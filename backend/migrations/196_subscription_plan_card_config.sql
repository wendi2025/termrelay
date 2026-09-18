ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS card_tier VARCHAR(30) NOT NULL DEFAULT 'standard',
    ADD COLUMN IF NOT EXISTS card_badge VARCHAR(80) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS card_featured BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS card_footnote TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS seat_limit INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS concurrency_limit INT NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS purchase_policy VARCHAR(20) NOT NULL DEFAULT 'public';

UPDATE subscription_plans
SET seat_limit = 1
WHERE seat_limit IS NULL OR seat_limit < 1;

UPDATE subscription_plans
SET concurrency_limit = 5
WHERE concurrency_limit IS NULL OR concurrency_limit < 1;

CREATE INDEX IF NOT EXISTS idx_subscription_plans_purchase_policy
    ON subscription_plans(purchase_policy);
