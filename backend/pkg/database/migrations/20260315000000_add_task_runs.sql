-- +goose Up
-- +goose StatementBegin

CREATE TABLE task_run (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    kind VARCHAR(50) NOT NULL,
    task_name VARCHAR(120) NOT NULL,
    status VARCHAR(50) NOT NULL,
    organization_id UUID REFERENCES organization(id) ON DELETE CASCADE,
    user_id UUID REFERENCES account(id) ON DELETE CASCADE,
    triggered_by_user_id UUID REFERENCES account(id) ON DELETE SET NULL,
    trigger_source VARCHAR(50) NOT NULL DEFAULT 'manual',
    parent_run_id UUID REFERENCES task_run(id) ON DELETE SET NULL,
    summary TEXT,
    error_summary TEXT,
    input_payload JSONB DEFAULT '{}',
    output_summary JSONB DEFAULT '{}',
    metrics JSONB DEFAULT '{}',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE task_run_step (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_run_id UUID NOT NULL REFERENCES task_run(id) ON DELETE CASCADE,
    step_key VARCHAR(120) NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    summary TEXT,
    error_summary TEXT,
    metrics JSONB DEFAULT '{}',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(task_run_id, step_key)
);

CREATE TABLE task_run_event (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_run_id UUID NOT NULL REFERENCES task_run(id) ON DELETE CASCADE,
    task_run_step_id UUID REFERENCES task_run_step(id) ON DELETE CASCADE,
    event_type VARCHAR(120) NOT NULL,
    level VARCHAR(20) NOT NULL DEFAULT 'info',
    message TEXT NOT NULL,
    meta_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_task_run_created ON task_run(created_at DESC);
CREATE INDEX idx_task_run_task_name_created ON task_run(task_name, created_at DESC);
CREATE INDEX idx_task_run_status_created ON task_run(status, created_at DESC);
CREATE INDEX idx_task_run_org_created ON task_run(organization_id, created_at DESC);
CREATE INDEX idx_task_run_user_created ON task_run(user_id, created_at DESC);
CREATE INDEX idx_task_run_parent_run_id ON task_run(parent_run_id);

CREATE INDEX idx_task_run_step_run_created ON task_run_step(task_run_id, created_at ASC);
CREATE INDEX idx_task_run_event_run_created ON task_run_event(task_run_id, created_at ASC);
CREATE INDEX idx_task_run_event_step_created ON task_run_event(task_run_step_id, created_at ASC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_task_run_event_step_created;
DROP INDEX IF EXISTS idx_task_run_event_run_created;
DROP INDEX IF EXISTS idx_task_run_step_run_created;
DROP INDEX IF EXISTS idx_task_run_parent_run_id;
DROP INDEX IF EXISTS idx_task_run_user_created;
DROP INDEX IF EXISTS idx_task_run_org_created;
DROP INDEX IF EXISTS idx_task_run_status_created;
DROP INDEX IF EXISTS idx_task_run_task_name_created;
DROP INDEX IF EXISTS idx_task_run_created;

DROP TABLE IF EXISTS task_run_event;
DROP TABLE IF EXISTS task_run_step;
DROP TABLE IF EXISTS task_run;

-- +goose StatementEnd
