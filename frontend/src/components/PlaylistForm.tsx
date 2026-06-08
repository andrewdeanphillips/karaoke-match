import { useState } from "react";
import type { SubmitEvent } from "react";

type PlaylistFormProps = {
  onSubmit: (url: string) => void;
  disabled: boolean;
};

function PlaylistForm({ onSubmit, disabled }: PlaylistFormProps) {
  const [url, setUrl] = useState("");

  function handleSubmit(event: SubmitEvent) {
    event.preventDefault();
    onSubmit(url);
  }

  return (
    <form onSubmit={handleSubmit}>
      <label htmlFor="playlist-url">Spotify playlist URL</label>
      <input
        id="playlist-url"
        type="text"
        value={url}
        onChange={(event) => setUrl(event.target.value)}
        disabled={disabled}
      />
      <button type="submit" disabled={disabled}>
        Check availability
      </button>
    </form>
  );
}

export default PlaylistForm;
