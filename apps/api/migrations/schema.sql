-- The schema the migrations add up to, in one place to read. The migrations
-- are what runs: a migration that changes the schema updates this file too.

CREATE TABLE links (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code       text        NOT NULL UNIQUE,
    url        text        NOT NULL,
    clicks     bigint      NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX links_created_at_idx ON links (created_at DESC);

CREATE TABLE admins (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      text        NOT NULL UNIQUE,
    -- bcrypt's output, which holds its own salt and cost; never the password.
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);
