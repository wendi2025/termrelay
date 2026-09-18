-- Persist one terminal outcome for every forwarded request, including failures
-- and client cancellations. Historical rows only contained completed requests.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'success',
    ADD COLUMN IF NOT EXISTS error_type VARCHAR(64),
    ADD COLUMN IF NOT EXISTS ended_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_usage_logs_status_created_at
    ON usage_logs (status, created_at);
