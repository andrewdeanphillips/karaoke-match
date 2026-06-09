export type AvailabilityResult = {
  artist: string;
  available: boolean;
  joysoundUrl?: string;
};

export type MatchResult = {
  results: AvailabilityResult[];
  totalArtists: number;
};
