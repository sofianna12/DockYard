-- database/init/002_create_projects.sql

CREATE TABLE projects (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         VARCHAR(255) NOT NULL,
    description   TEXT,
    repository    VARCHAR(500),
    docker_image       VARCHAR(500) NOT NULL,
    registry_user      VARCHAR(255) NOT NULL DEFAULT '',
    registry_password  VARCHAR(255) NOT NULL DEFAULT '',
    env_vars           TEXT NOT NULL DEFAULT '',
    auto_stop_min INTEGER NOT NULL DEFAULT 60,
    status        VARCHAR(50) NOT NULL DEFAULT 'stopped',
    container_id  VARCHAR(255),
    port          INTEGER,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_projects_user_id ON projects(user_id);
CREATE INDEX idx_projects_status  ON projects(status);
