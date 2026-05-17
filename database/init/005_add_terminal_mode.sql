-- database/init/005_add_terminal_mode.sql

ALTER TABLE projects
    ADD COLUMN terminal_mode BOOLEAN NOT NULL DEFAULT FALSE;
