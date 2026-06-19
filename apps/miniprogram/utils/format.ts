export function formatScore(score: number): string {
  return score > 0 ? `+${score}` : `${score}`;
}

export function toInteger(value: string): number | null {
  if (!/^-?\d+$/.test(value.trim())) return null;
  return Number(value);
}
