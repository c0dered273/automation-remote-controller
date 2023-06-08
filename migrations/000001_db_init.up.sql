CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users
(
    id       int GENERATED ALWAYS AS IDENTITY,
    username varchar(32) UNIQUE NOT NULL,
    password varchar(72)        NOT NULL,
    tg_user  varchar(32) UNIQUE NOT NULL,
    chat_id BIGINT UNIQUE,
    notify_enabled boolean NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS clients
(
    id      int GENERATED ALWAYS AS IDENTITY,
    name    varchar(64) UNIQUE NOT NULL,
    uuid    varchar(36) UNIQUE NOT NULL,
    user_id int                NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_users FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
