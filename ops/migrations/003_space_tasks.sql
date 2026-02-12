CREATE TABLE space_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_space_id UUID NOT NULL REFERENCES bot_spaces (id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'available',
    bot_id UUID REFERENCES bots (id) ON DELETE SET NULL,
    created_by_bot_id UUID NOT NULL REFERENCES bots (id) ON DELETE CASCADE,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_space_tasks_space_status ON space_tasks (bot_space_id, status);
