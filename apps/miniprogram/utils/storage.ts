const TOKEN_KEY = 'one_round_token';
const USER_KEY = 'one_round_user';
const RECENT_KEY = 'one_round_recent_sessions';

export type UserInfo = {
  id: string;
  displayName: string | null;
  avatarUrl: string | null;
};

export function getToken(): string {
  return wx.getStorageSync(TOKEN_KEY) || '';
}

export function setToken(token: string): void {
  wx.setStorageSync(TOKEN_KEY, token);
}

export function clearToken(): void {
  wx.removeStorageSync(TOKEN_KEY);
}

export function getUser(): UserInfo | null {
  return wx.getStorageSync(USER_KEY) || null;
}

export function setUser(user: UserInfo): void {
  wx.setStorageSync(USER_KEY, user);
}

export function clearUser(): void {
  wx.removeStorageSync(USER_KEY);
}

export function saveRecentSession(gameSessionId: string): void {
  const list = getRecentSessions().filter((id) => id !== gameSessionId);
  wx.setStorageSync(RECENT_KEY, [gameSessionId, ...list].slice(0, 20));
}

export function getRecentSessions(): string[] {
  return wx.getStorageSync(RECENT_KEY) || [];
}
