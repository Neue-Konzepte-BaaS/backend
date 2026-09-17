-- name: InsertBroadcastNotification :one
INSERT INTO broadcast_notification (subject, body)
VALUES ($1, $2)
RETURNING id, subject, body, created_at;

-- name: GetAllBroadcastNotifications :many
SELECT id, subject, body, created_at
FROM broadcast_notification
ORDER BY created_at DESC;
