-- database/init/006_add_web_terminal.sql

ALTER TABLE projects
    ADD COLUMN web_terminal BOOLEAN NOT NULL DEFAULT FALSE;
