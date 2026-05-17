-- database/init/003_add_lifecycle.sql

ALTER TABLE projects
    ADD COLUMN started_at       TIMESTAMP WITH TIME ZONE,
    ADD COLUMN last_accessed_at TIMESTAMP WITH TIME ZONE;
