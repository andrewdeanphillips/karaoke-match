import MatchPage from "./pages/MatchPage";
import SpotifySessionGate from "./components/SpotifySessionGate";

function App() {
  return (
    <SpotifySessionGate>
      <MatchPage />
    </SpotifySessionGate>
  );
}

export default App;
