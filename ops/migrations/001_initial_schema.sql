-- 001_initial_schema.sql
-- Initial database schema for Claw-Swarm API

BEGIN;

-- users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL CHECK (email = lower(email)),
    password_hash TEXT NOT NULL,
    display_name VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- bot_spaces
CREATE TABLE bot_spaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    join_code VARCHAR(64) UNIQUE NOT NULL,
    manager_join_code VARCHAR(64) UNIQUE NOT NULL,
    manager_bot_id UUID,  -- FK added after bots table
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- space_members
CREATE TABLE space_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_space_id UUID NOT NULL REFERENCES bot_spaces (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bot_space_id, user_id)
);

-- bots
CREATE TABLE bots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_space_id UUID NOT NULL REFERENCES bot_spaces (id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    capabilities TEXT,
    is_manager BOOLEAN NOT NULL DEFAULT false,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Now add the deferred FK from bot_spaces to bots
ALTER TABLE bot_spaces
ADD CONSTRAINT fk_bot_spaces_manager_bot
FOREIGN KEY (manager_bot_id) REFERENCES bots (id) ON DELETE SET NULL;

-- messages
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_space_id UUID NOT NULL REFERENCES bot_spaces (id) ON DELETE CASCADE,
    sender_id UUID NOT NULL,
    sender_name VARCHAR(100) NOT NULL,
    sender_type TEXT NOT NULL CHECK (sender_type IN ('bot', 'user')),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- bot_statuses
CREATE TABLE bot_statuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_space_id UUID NOT NULL REFERENCES bot_spaces (id) ON DELETE CASCADE,
    bot_id UUID NOT NULL REFERENCES bots (id) ON DELETE CASCADE,
    bot_name VARCHAR(100) NOT NULL,
    status TEXT NOT NULL,
    updated_by_bot_id UUID NOT NULL REFERENCES bots (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bot_space_id, bot_id)
);

-- summaries
CREATE TABLE summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_space_id UUID UNIQUE NOT NULL REFERENCES bot_spaces (
        id
    ) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_by_bot_id UUID NOT NULL REFERENCES bots (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- invite_codes
CREATE TABLE invite_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_space_id UUID NOT NULL REFERENCES bot_spaces (id) ON DELETE CASCADE,
    code VARCHAR(64) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX idx_messages_space_created ON messages (
    bot_space_id, created_at DESC
);
CREATE INDEX idx_bots_space ON bots (bot_space_id);
CREATE INDEX idx_bot_statuses_space ON bot_statuses (bot_space_id);
CREATE INDEX idx_invite_codes_space ON invite_codes (bot_space_id);

COMMIT;
