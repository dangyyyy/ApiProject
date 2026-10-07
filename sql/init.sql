DROP TABLE IF EXISTS tasks;

CREATE TABLE tasks (
                       id          SERIAL PRIMARY KEY,
                       title       VARCHAR(255) NOT NULL,
                       description TEXT         NOT NULL DEFAULT '',
                       completed   BOOLEAN      NOT NULL DEFAULT false,
                       created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                       updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

INSERT INTO tasks (title, description, completed) VALUES
                                                      ('Изучить Go', 'Написать REST API', false),
                                                      ('Написать REST', 'Посмотреть видео', true);