import { useState } from "react";
import { matchPlaylist, MatchError } from "../api/match";
import type { MatchResult } from "../types/match";
import PlaylistForm from "../components/PlaylistForm";
import SummaryStats from "../components/SummaryStats";
import ResultsTable from "../components/ResultsTable";

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
      } else if (err instanceof MatchError && err.status === 401) {
        setError("Your Spotify session has expired — refresh the page to log in again.");
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
        <>
          <SummaryStats result={result} />
          <ResultsTable results={result.results} />
        </>
      )}
    </main>
  );
}

export default MatchPage;
