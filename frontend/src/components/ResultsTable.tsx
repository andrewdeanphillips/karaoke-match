import type { AvailabilityResult } from "../types/match";

type ResultsTableProps = {
  results: AvailabilityResult[];
};

function ResultsTable({ results }: ResultsTableProps) {
  return (
    <table>
      <thead>
        <tr>
          <th>Artist</th>
          <th>JOYSOUND</th>
        </tr>
      </thead>
      <tbody>
        {results.map((result) => (
          <tr key={result.artist}>
            <td>{result.artist}</td>
            <td>{result.available ? "Yes" : "No"}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export default ResultsTable;
