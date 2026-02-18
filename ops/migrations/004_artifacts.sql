CREATE TABLE artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_space_id UUID NOT NULL REFERENCES bot_spaces (id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    data TEXT NOT NULL,
    created_by_bot_id UUID NOT NULL REFERENCES bots (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_artifacts_bot_space ON artifacts (bot_space_id);
