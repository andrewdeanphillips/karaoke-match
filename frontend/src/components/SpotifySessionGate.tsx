import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import { hasSpotifySession, loginURL } from "../api/auth";

type SessionStatus = "checking" | "loggedIn" | "loggedOut";

type SpotifySessionGateProps = {
  children: ReactNode;
};

// SpotifySessionGate only renders its children once the visitor has a usable
// Spotify session. KaraokeMatch reads playlists from the visitor's own
// account, so there's nothing the rest of the app can do without one —
// visitors who aren't logged in see a prompt to log in with Spotify instead.
function SpotifySessionGate({ children }: SpotifySessionGateProps) {
  const [status, setStatus] = useState<SessionStatus>("checking");

  useEffect(() => {
    let cancelled = false;

    hasSpotifySession()
      .then((loggedIn) => {
        if (!cancelled) setStatus(loggedIn ? "loggedIn" : "loggedOut");
      })
      .catch(() => {
        if (!cancelled) setStatus("loggedOut");
      });

    return () => {
      cancelled = true;
    };
  }, []);

  if (status === "checking") {
    return (
      <main>
        <h1>KaraokeMatch</h1>
        <p>Checking your Spotify session…</p>
      </main>
    );
  }

  if (status === "loggedOut") {
    return (
      <main>
        <h1>KaraokeMatch</h1>
        <p>
          KaraokeMatch reads playlists from your own Spotify account, so
          you'll need to log in to get started.
        </p>
        <a href={loginURL}>Log in with Spotify</a>
      </main>
    );
  }

  return <>{children}</>;
}

export default SpotifySessionGate;
