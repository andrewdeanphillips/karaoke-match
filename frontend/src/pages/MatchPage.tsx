import { useState } from "react";
import { matchPlaylist, MatchError } from "../api/match";
import type { MatchResult } from "../types/match";
import PlaylistForm from "../components/PlaylistForm";

function MatchPage() {
  const [result, setResult] = useState<MatchResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(url: string) {
    setError(null);
    setResult(null);
    setLoading(true);

    try {
      setResult(await matchPlaylist(url));
    } catch (err) {
      if (err instanceof MatchError && err.status === 400) {
        setError(err.message);
      } else {
        setError("Something went wrong on our end — please try again in a moment.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <main>
      <h1>KaraokeMatch</h1>
      <PlaylistForm onSubmit={handleSubmit} disabled={loading} />
      {loading && <p>Checking…</p>}
      {error && <p>Error: {error}</p>}
      {result && (
        <ul>
          {result.results.map((r) => (
            <li key={r.artist}>
              {r.artist}: {r.available ? "available" : "not available"}
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}

export default MatchPage;
