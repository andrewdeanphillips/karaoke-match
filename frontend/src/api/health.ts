import type { HealthStatus } from "../types/health";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL;

export async function getHealth(): Promise<HealthStatus> {
  const response = await fetch(`${API_BASE_URL}/health`);

  if (!response.ok) {
    throw new Error(`Health check failed: ${response.status}`);
  }

  return response.json();
}
