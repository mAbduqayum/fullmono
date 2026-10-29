CREATE TABLE admins (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      text        NOT NULL UNIQUE,
    -- bcrypt's output, which holds its own salt and cost; never the password.
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);
