import { useEffect, useState } from "react";
import { getHealth } from "./api/health";
import type { HealthStatus } from "./types/health";

function App() {
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadHealth() {
      try {
        const result = await getHealth();
        setHealth(result);
      } catch (err) {
        setError((err as Error).message);
      }
    }

    loadHealth();
  }, []);

  return (
    <main>
      <h1>KaraokeMatch</h1>
      <h2>Backend status</h2>
      {error && <p>Error: {error}</p>}
      {!error && !health && <p>Checking backend…</p>}
      {health && (
        <ul>
          <li>API: {health.status}</li>
          <li>Database: {health.database}</li>
        </ul>
      )}
    </main>
  );
}

export default App;
