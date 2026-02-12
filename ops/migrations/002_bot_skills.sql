BEGIN;

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE bot_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_space_id UUID NOT NULL REFERENCES bot_spaces(id) ON DELETE CASCADE,
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    bot_name VARCHAR(100) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    tags VARCHAR(30)[],
    embedding vector(1536) DEFAULT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bot_space_id, bot_id, name)
);

CREATE INDEX idx_bot_skills_space ON bot_skills(bot_space_id);

COMMIT;
