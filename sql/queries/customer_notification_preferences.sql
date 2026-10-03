-- name: GetCustomerNotificationPreferences :one
SELECT notify_messages_by_email
FROM customer
WHERE account_id = $1;

-- name: UpdateCustomerNotificationPreferences :execrows
UPDATE customer
SET notify_messages_by_email = $2
WHERE account_id = $1;
