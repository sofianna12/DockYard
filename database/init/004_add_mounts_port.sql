-- database/init/004_add_mounts_port.sql

ALTER TABLE projects
    ADD COLUMN mounts         TEXT         NOT NULL DEFAULT '',
    ADD COLUMN container_port INTEGER      NOT NULL DEFAULT 0,
    ADD COLUMN scheme         VARCHAR(10)  NOT NULL DEFAULT 'http';
