export type AvailabilityResult = {
  artist: string;
  available: boolean;
};

export type MatchResult = {
  results: AvailabilityResult[];
  totalArtists: number;
};
