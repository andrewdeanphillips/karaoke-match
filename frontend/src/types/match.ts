export type TrackAvailabilityResult = {
  artist: string;
  title: string;
  artistJoysoundUrl?: string;
  songJoysoundUrl?: string;
};

export type MatchResult = {
  results: TrackAvailabilityResult[];
  totalTracks: number;
};
