CREATE TABLE IF NOT EXISTS code_activity (
    repository VARCHAR CHECK (length(repository) < 128),
    workspace VARCHAR CHECK (length(workspace) < 64),
    "filename" VARCHAR CHECK (length("filename") < 64),
    "language" VARCHAR CHECK (length("language") < 32),
    "row" SMALLINT CHECK ("row" >= 0),
    "column" SMALLINT CHECK ("column" >= 0),
    code_chunk VARCHAR,
    reported_at TIMESTAMPTZ
);