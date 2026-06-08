CREATE TABLE artist_availability (
    id                BIGSERIAL PRIMARY KEY,
    artist            TEXT NOT NULL,
    catalog           TEXT NOT NULL,
    available         BOOLEAN NOT NULL,
    catalog_artist_id TEXT,
    last_checked      TIMESTAMPTZ NOT NULL,
    UNIQUE (artist, catalog)
);
