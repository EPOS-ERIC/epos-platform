-- name: ListNotificationEventTypes :many
SELECT
    KEY,
    scope,
    MODE,
    default_on
FROM
    notification_event_types
ORDER BY
    KEY;

-- name: ListNotificationEventTypePermissions :many
SELECT
    permission_key
FROM
    notification_event_type_permissions
WHERE
    event_type = sqlc.arg(event_type)
ORDER BY
    permission_key;

-- name: ListGlobalNotificationSubscriptions :many
SELECT
    subscription.user_id,
    subscription.event_type,
    subscription.channel
FROM
    global_notification_subscriptions AS subscription
WHERE
    subscription.user_id = sqlc.arg(user_id)
ORDER BY
    subscription.event_type,
    subscription.channel;

-- name: ListGroupNotificationSubscriptions :many
SELECT
    subscription.user_id,
    subscription.group_id,
    subscription.event_type,
    subscription.channel
FROM
    group_notification_subscriptions AS subscription
WHERE
    subscription.user_id = sqlc.arg(user_id)
ORDER BY
    subscription.group_id,
    subscription.event_type,
    subscription.channel;

-- name: GetNotificationUser :one
SELECT
    id,
    email,
    STATUS
FROM
    users
WHERE
    id = sqlc.arg(user_id);

-- name: ListGlobalNotificationRecipients :many
SELECT
    DISTINCT users.id,
    users.email,
    subscription.channel
FROM
    global_notification_subscriptions AS subscription
    JOIN users ON users.id = subscription.user_id
WHERE
    subscription.event_type = sqlc.arg(event_type)
    AND users.status = 'ACTIVE'
    AND (
        NOT EXISTS (
            SELECT
                1
            FROM
                notification_event_type_permissions AS required
            WHERE
                required.event_type = subscription.event_type
        )
        OR EXISTS (
            SELECT
                1
            FROM
                global_permission_grants AS permission_grant
            WHERE
                permission_grant.user_id = users.id
                AND permission_grant.permission_key IN (
                    SELECT
                        required.permission_key
                    FROM
                        notification_event_type_permissions AS required
                    WHERE
                        required.event_type = subscription.event_type
                )
        )
        OR EXISTS (
            SELECT
                1
            FROM
                user_global_roles AS role
                JOIN role_permissions AS role_permission ON role_permission.role_key = role.role_key
            WHERE
                role.user_id = users.id
                AND role_permission.permission_key IN (
                    SELECT
                        required.permission_key
                    FROM
                        notification_event_type_permissions AS required
                    WHERE
                        required.event_type = subscription.event_type
                )
        )
    )
ORDER BY
    users.id;

-- name: ListGroupNotificationRecipients :many
SELECT
    DISTINCT users.id,
    users.email,
    subscription.channel
FROM
    group_notification_subscriptions AS subscription
    JOIN users ON users.id = subscription.user_id
    JOIN group_memberships AS membership ON membership.user_id = subscription.user_id
    AND membership.group_id = subscription.group_id
WHERE
    subscription.group_id = sqlc.arg(group_id)
    AND subscription.event_type = sqlc.arg(event_type)
    AND users.status = 'ACTIVE'
    AND membership.status = 'ACTIVE'
    AND (
        NOT EXISTS (
            SELECT
                1
            FROM
                notification_event_type_permissions AS required
            WHERE
                required.event_type = subscription.event_type
        )
        OR EXISTS (
            SELECT
                1
            FROM
                group_permission_grants AS permission_grant
            WHERE
                permission_grant.user_id = users.id
                AND permission_grant.group_id = subscription.group_id
                AND permission_grant.permission_key IN (
                    SELECT
                        required.permission_key
                    FROM
                        notification_event_type_permissions AS required
                    WHERE
                        required.event_type = subscription.event_type
                )
        )
        OR EXISTS (
            SELECT
                1
            FROM
                group_membership_roles AS role
                JOIN role_permissions AS role_permission ON role_permission.role_key = role.role_key
            WHERE
                role.user_id = users.id
                AND role.group_id = subscription.group_id
                AND role_permission.permission_key IN (
                    SELECT
                        required.permission_key
                    FROM
                        notification_event_type_permissions AS required
                    WHERE
                        required.event_type = subscription.event_type
                )
        )
    )
ORDER BY
    users.id;

-- name: CreateGlobalNotificationSubscription :one
INSERT INTO
    global_notification_subscriptions (user_id, event_type, channel)
VALUES
    (
        sqlc.arg(user_id),
        sqlc.arg(event_type),
        sqlc.arg(channel)
    ) ON CONFLICT (user_id, event_type, channel) DO
UPDATE
SET
    event_type = EXCLUDED.event_type
RETURNING
    user_id,
    event_type,
    channel;

-- name: CreateGroupNotificationSubscription :one
INSERT INTO
    group_notification_subscriptions (user_id, group_id, event_type, channel)
VALUES
    (
        sqlc.arg(user_id),
        sqlc.arg(group_id),
        sqlc.arg(event_type),
        sqlc.arg(channel)
    ) ON CONFLICT (user_id, group_id, event_type, channel) DO
UPDATE
SET
    event_type = EXCLUDED.event_type
RETURNING
    user_id,
    group_id,
    event_type,
    channel;

-- name: DeleteGlobalNotificationSubscription :execrows
DELETE FROM
    global_notification_subscriptions
WHERE
    user_id = sqlc.arg(user_id)
    AND event_type = sqlc.arg(event_type)
    AND channel = sqlc.arg(channel);

-- name: DeleteGroupNotificationSubscription :execrows
DELETE FROM
    group_notification_subscriptions
WHERE
    user_id = sqlc.arg(user_id)
    AND group_id = sqlc.arg(group_id)
    AND event_type = sqlc.arg(event_type)
    AND channel = sqlc.arg(channel);
