CREATE TABLE song_availability (
    id              BIGSERIAL PRIMARY KEY,
    track_artist    TEXT NOT NULL,
    track_title     TEXT NOT NULL,
    catalog         TEXT NOT NULL,
    available       BOOLEAN NOT NULL,
    catalog_song_id TEXT,
    last_checked    TIMESTAMPTZ NOT NULL,
    UNIQUE (track_artist, track_title, catalog)
);
