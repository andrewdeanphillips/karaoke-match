package karaoke

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// cacheEntry is a cached availability record for one artist on one catalog.
// CatalogArtistID is empty when the artist wasn't found on the catalog —
// there's no ID to anchor to in that case.
type cacheEntry struct {
	Available       bool
	CatalogArtistID string
	LastChecked     time.Time
}

// cache is the subset of postgresCache that Service depends on — narrow
// enough that tests can substitute a fake and exercise caching/staleness
// logic without making real database queries.
type cache interface {
	lookup(ctx context.Context, catalog, artist string) (cacheEntry, bool, error)
	lookupBatch(ctx context.Context, catalog string, artists []string) (map[string]cacheEntry, error)
	store(ctx context.Context, catalog, artist string, entry cacheEntry) error
}

type postgresCache struct {
	pool *pgxpool.Pool
}

func newPostgresCache(pool *pgxpool.Pool) *postgresCache {
	return &postgresCache{pool: pool}
}

func (c *postgresCache) lookup(ctx context.Context, catalog, artist string) (cacheEntry, bool, error) {
	var entry cacheEntry
	var catalogArtistID *string

	err := c.pool.QueryRow(ctx,
		`SELECT available, catalog_artist_id, last_checked
		 FROM artist_availability
		 WHERE artist = $1 AND catalog = $2`,
		artist, catalog,
	).Scan(&entry.Available, &catalogArtistID, &entry.LastChecked)

	if errors.Is(err, pgx.ErrNoRows) {
		return cacheEntry{}, false, nil
	}
	if err != nil {
		return cacheEntry{}, false, fmt.Errorf("looking up cached availability for %q on %q: %w", artist, catalog, err)
	}

	if catalogArtistID != nil {
		entry.CatalogArtistID = *catalogArtistID
	}
	return entry, true, nil
}

// lookupBatch returns cached availability records for whichever of the given
// artists have one, in a single round trip — the multi-artist counterpart to
// lookup, for callers (like CheckAvailability) that need to check many
// artists at once rather than paying one round trip per artist. Artists with
// no cached record simply have no entry in the returned map; this mirrors
// lookup's (cacheEntry{}, false, nil) "not found" signal, just batched.
func (c *postgresCache) lookupBatch(ctx context.Context, catalog string, artists []string) (map[string]cacheEntry, error) {
	rows, err := c.pool.Query(ctx,
		`SELECT artist, available, catalog_artist_id, last_checked
		 FROM artist_availability
		 WHERE catalog = $1 AND artist = ANY($2)`,
		catalog, artists,
	)
	if err != nil {
		return nil, fmt.Errorf("looking up cached availability for %d artists on %q: %w", len(artists), catalog, err)
	}
	defer rows.Close()

	entries := make(map[string]cacheEntry)
	for rows.Next() {
		var artist string
		var entry cacheEntry
		var catalogArtistID *string

		if err := rows.Scan(&artist, &entry.Available, &catalogArtistID, &entry.LastChecked); err != nil {
			return nil, fmt.Errorf("scanning cached availability row for %q: %w", catalog, err)
		}
		if catalogArtistID != nil {
			entry.CatalogArtistID = *catalogArtistID
		}
		entries[artist] = entry
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading cached availability rows for %q: %w", catalog, err)
	}

	return entries, nil
}

// store overwrites any existing record for the given artist and catalog, so
// repeat checks refresh the cache rather than accumulating stale duplicates.
func (c *postgresCache) store(ctx context.Context, catalog, artist string, entry cacheEntry) error {
	var catalogArtistID *string
	if entry.CatalogArtistID != "" {
		catalogArtistID = &entry.CatalogArtistID
	}

	_, err := c.pool.Exec(ctx,
		`INSERT INTO artist_availability (artist, catalog, available, catalog_artist_id, last_checked)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (artist, catalog) DO UPDATE SET
		     available = excluded.available,
		     catalog_artist_id = excluded.catalog_artist_id,
		     last_checked = excluded.last_checked`,
		artist, catalog, entry.Available, catalogArtistID, entry.LastChecked,
	)
	if err != nil {
		return fmt.Errorf("storing availability for %q on %q: %w", artist, catalog, err)
	}
	return nil
}
