import { useState } from "react";
import type { FormEvent } from "react";

type PlaylistFormProps = {
  onSubmit: (url: string) => void;
  disabled: boolean;
};

function PlaylistForm({ onSubmit, disabled }: PlaylistFormProps) {
  const [url, setUrl] = useState("");

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSubmit(url);
  }

  return (
    <form className="playlist-form" onSubmit={handleSubmit}>
      <div>
        <label htmlFor="playlist-url">Spotify playlist URL</label>
        <input
          id="playlist-url"
          type="text"
          placeholder="https://open.spotify.com/playlist/…"
          value={url}
          onChange={(event) => setUrl(event.target.value)}
          disabled={disabled}
        />
      </div>
      <button type="submit" disabled={disabled}>
        Check availability
      </button>
    </form>
  );
}

export default PlaylistForm;
