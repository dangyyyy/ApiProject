DROP TABLE IF EXISTS tasks;

CREATE TABLE tasks (
                       id          SERIAL PRIMARY KEY,
                       title       VARCHAR(255) NOT NULL,
                       description TEXT         NOT NULL DEFAULT '',
                       completed   BOOLEAN      NOT NULL DEFAULT false,
                       created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                       updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_tasks_created_at ON tasks (created_at DESC, id DESC);
CREATE INDEX idx_tasks_completed_created_at ON tasks (completed, created_at DESC, id DESC);
INSERT INTO tasks (title, description, completed) VALUES
                                                      ('Изучить Go', 'Написать REST API', false),
                                                      ('Написать REST', 'Посмотреть видео', true);