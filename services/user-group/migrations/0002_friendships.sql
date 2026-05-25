BEGIN;

CREATE TABLE IF NOT EXISTS friendships (
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    friend_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, friend_id),
    CONSTRAINT friendships_self_check CHECK (user_id <> friend_id)
);

CREATE INDEX IF NOT EXISTS friendships_friend_idx ON friendships (friend_id);

COMMIT;
