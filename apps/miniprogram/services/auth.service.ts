import type { User } from '../models/user';
import { request } from './http';
import { setToken } from '../utils/storage';

type LoginResponse = {
  token: string;
  user: User;
};

export async function login(): Promise<void> {
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
}
