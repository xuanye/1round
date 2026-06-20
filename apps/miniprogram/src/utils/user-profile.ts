export function isSystemDisplayName(displayName: string | null | undefined): boolean {
  return /^老书记\d{2}$/.test(String(displayName || '').trim());
}
