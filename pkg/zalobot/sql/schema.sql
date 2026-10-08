-- PostgreSQL. Đổi tên bảng/khóa ngoại cho khớp dự án đích:
--   owner_id  -> chủ bot (manager, shop...)
--   target_id -> đối tượng gắn với nhóm Zalo (room, order...)
-- uuidv7() cần PostgreSQL 18; bản cũ hơn dùng gen_random_uuid().

CREATE TABLE IF NOT EXISTS zalo_bot_configs (
    owner_id                 uuid PRIMARY KEY,
    encrypted_bot_token      text NOT NULL,
    encrypted_webhook_secret text NOT NULL,
    is_active                boolean NOT NULL DEFAULT true,
    owner_zalo_user_id       varchar(255),
    created_at               timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at               timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS zalo_user_links (
    user_id      uuid PRIMARY KEY,
    zalo_user_id varchar(255) NOT NULL UNIQUE,
    created_at   timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS zalo_group_links (
    id            uuid PRIMARY KEY DEFAULT uuidv7(),
    owner_id      uuid NOT NULL,
    target_id     uuid NOT NULL UNIQUE,
    group_chat_id varchar(255) NOT NULL UNIQUE,
    created_at    timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_zalo_group_links_owner_id ON zalo_group_links (owner_id);
