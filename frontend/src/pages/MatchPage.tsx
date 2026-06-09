import { useState, useEffect } from "react";
import { matchPlaylist, MatchError } from "../api/match";
import type { MatchResult } from "../types/match";
import PlaylistForm from "../components/PlaylistForm";
import SummaryStats from "../components/SummaryStats";
import ResultsList from "../components/ResultsList";

function MatchPage() {
  const [result, setResult] = useState<MatchResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [slowLoad, setSlowLoad] = useState(false);

  useEffect(() => {
    if (!loading) {
      setSlowLoad(false);
      return;
    }
    const timer = setTimeout(() => setSlowLoad(true), 3000);
    return () => clearTimeout(timer);
  }, [loading]);

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
    <main className="page">
      <h1>KaraokeMatch</h1>
      <p className="intro">
        Paste one of your Spotify playlist URLs to see which of its artists
        are available on JOYSOUND.
      </p>
      <PlaylistForm onSubmit={handleSubmit} disabled={loading} />
      {loading && (
        <p className="status">
          {slowLoad
            ? "Still checking — JOYSOUND lookups are paced to avoid hammering their servers, so a large playlist can take up to 10 seconds."
            : "Checking…"}
        </p>
      )}
      {error && <p className="status status--error">{error}</p>}
      {result && (
        <div className="section">
          <SummaryStats result={result} />
          <ResultsList results={result.results} />
        </div>
      )}
    </main>
  );
}

export default MatchPage;
