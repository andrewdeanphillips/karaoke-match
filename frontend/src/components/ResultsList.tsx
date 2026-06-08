import { useState } from "react";
import type { AvailabilityResult } from "../types/match";

type ResultsListProps = {
  results: AvailabilityResult[];
};

// ResultsList splits matched artists into "available" and "not found" groups
// — the grouping itself is the headline information, and it keeps a long
// playlist scannable in a way a flat available/unavailable column never was.
// The search box narrows both groups at once, which matters most for exactly
// the playlists where scanning by eye stops being practical.
function ResultsList({ results }: ResultsListProps) {
  const [query, setQuery] = useState("");

  const needle = query.trim().toLowerCase();
  const matching = needle
    ? results.filter((result) => result.artist.toLowerCase().includes(needle))
    : results;

  const available = matching.filter((result) => result.available);
  const notFound = matching.filter((result) => !result.available);

  return (
    <div className="results">
      <input
        type="search"
        className="results-search"
        placeholder="Search artists…"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        aria-label="Search artists"
      />
      <div className="results-columns">
        <ResultsColumn label="Available" artists={available} variant="available" />
        <ResultsColumn label="Not found" artists={notFound} variant="not-found" />
      </div>
    </div>
  );
}

type ResultsColumnProps = {
  label: string;
  artists: AvailabilityResult[];
  variant: "available" | "not-found";
};

function ResultsColumn({ label, artists, variant }: ResultsColumnProps) {
  return (
    <div className={`results-column results-column--${variant}`}>
      <h3 className="results-column-heading">
        {label} <span className="count">{artists.length}</span>
      </h3>
      {artists.length > 0 ? (
        <ul>
          {artists.map((result) => (
            <li key={result.artist}>{result.artist}</li>
          ))}
        </ul>
      ) : (
        <p className="results-empty">No matches</p>
      )}
    </div>
  );
}

export default ResultsList;
