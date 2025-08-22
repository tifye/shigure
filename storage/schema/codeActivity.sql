CREATE TABLE IF NOT EXISTS code_activity (
    repository STRING CHECK (length("language") < 128),
    workspace STRING CHECK (length("language") < 64),
    "filename" STRING CHECK (length("filename") < 64),
    "language" STRING CHECK (length("language") < 32),
    "row" SHORT CHECK ("row" >= 0),
    "column" SHORT CHECK ("column" >= 0),
    code_chunk STRING,
    reported_at TIMESTAMPTZ
);