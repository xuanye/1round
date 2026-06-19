import type { User } from '../models/user';
import { request } from './http';
import { setToken, getToken, setUser } from '../utils/storage';

type LoginResponse = {
  token: string;
  user: User;
};

let loginPromise: Promise<void> | null = null;

export async function login(): Promise<void> {
  if (loginPromise) return loginPromise;
  loginPromise = (async () => {
    try {
      const code = await new Promise<string>((resolve, reject) => {
        wx.login({
          success: (res) => resolve(res.code),
          fail: reject,
        });
      });
      const result = await request<LoginResponse>({
        url: '/api/auth/wechat-login',
        method: 'POST',
        data: { code },
        auth: false,
      });
      setToken(result.token);
      setUser({
        id: result.user.id,
        displayName: result.user.displayName,
        avatarUrl: result.user.avatarUrl,
      });
    } finally {
      loginPromise = null;
    }
  })();
  return loginPromise;
}

export async function ensureLogin(): Promise<void> {
  const token = getToken();
  if (token) return;
  await login();
}
