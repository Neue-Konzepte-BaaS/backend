-- migrate:up
CREATE TABLE broadcast_notification (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_broadcast_notification_created ON broadcast_notification(created_at DESC);

-- migrate:down
DROP TABLE broadcast_notification;
