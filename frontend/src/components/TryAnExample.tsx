import { useEffect, useState } from "react";
import { fetchExamples, matchExample } from "../api/examples";
import type { Example } from "../api/examples";
import type { MatchResult } from "../types/match";
import SummaryStats from "./SummaryStats";
import ResultsTable from "./ResultsTable";

// TryAnExample lets a visitor without a Spotify account see KaraokeMatch run
// end-to-end anyway — one click against a playlist Andrew picked himself,
// matched through his own session rather than the visitor's. It renders
// nothing once the example list comes back empty, which is what the backend
// reports when that path isn't configured.
function TryAnExample() {
  const [examples, setExamples] = useState<Example[]>([]);
  const [activeID, setActiveID] = useState<string | null>(null);
  const [result, setResult] = useState<MatchResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;

    fetchExamples().then((fetched) => {
      if (!cancelled) setExamples(fetched);
    });

    return () => {
      cancelled = true;
    };
  }, []);

  async function handleTry(example: Example) {
    setActiveID(example.id);
    setError(null);
    setResult(null);
    setLoading(true);

    try {
      setResult(await matchExample(example.id));
    } catch {
      setError("Something went wrong loading that example — please try again in a moment.");
    } finally {
      setLoading(false);
    }
  }

  if (examples.length === 0) {
    return null;
  }

  const active = examples.find((example) => example.id === activeID);

  return (
    <section>
      <h2>Try an example — no login required</h2>
      <p>See how KaraokeMatch handles a few playlists Andrew picked himself.</p>
      <ul>
        {examples.map((example) => (
          <li key={example.id}>
            <button onClick={() => handleTry(example)} disabled={loading}>
              {example.name}
            </button>
          </li>
        ))}
      </ul>
      {loading && <p>Checking {active?.name}…</p>}
      {error && <p>Error: {error}</p>}
      {result && (
        <>
          <SummaryStats result={result} />
          <ResultsTable results={result.results} />
        </>
      )}
    </section>
  );
}

export default TryAnExample;
