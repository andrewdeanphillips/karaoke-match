import { useState } from "react";
import { matchPlaylist } from "../api/match";
import type { MatchResult } from "../types/match";
import PlaylistForm from "../components/PlaylistForm";

function MatchPage() {
  const [result, setResult] = useState<MatchResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(url: string) {
    setError(null);
    setResult(null);

    try {
      setResult(await matchPlaylist(url));
    } catch (err) {
      setError((err as Error).message);
    }
  }

  return (
    <main>
      <h1>KaraokeMatch</h1>
      <PlaylistForm onSubmit={handleSubmit} disabled={false} />
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
