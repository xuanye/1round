import { clearToken, getToken } from '../utils/storage';

export type ApiResponse<T> = {
  code: number;
  message: string;
  data: T | null;
};

type Method = 'GET' | 'POST' | 'PATCH' | 'DELETE';

function baseUrl(): string {
  const app = getApp<{ globalData: { baseUrl: string } }>();
  return app.globalData.baseUrl;
}

export async function request<T>(options: {
  url: string;
  method?: Method;
  data?: unknown;
  auth?: boolean;
}): Promise<T> {
  return new Promise((resolve, reject) => {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (options.auth !== false) {
      const token = getToken();
      if (token) headers.Authorization = `Bearer ${token}`;
    }
    wx.request<ApiResponse<T>>({
      url: `${baseUrl()}${options.url}`,
      method: (options.method || 'GET') as WechatMiniprogram.RequestOption['method'],
      data: options.data as WechatMiniprogram.RequestOption['data'],
      header: headers,
      success(res) {
        if (res.statusCode === 401) {
          clearToken();
          wx.redirectTo({ url: '/pages/home/index' });
          reject(new Error('unauthorized'));
          return;
        }
        if (!res.data || res.data.code !== 0 || res.data.data === null) {
          reject(new Error(res.data?.message || 'request failed'));
          return;
        }
        resolve(res.data.data);
      },
      fail: reject,
    });
  });
}

export async function requestBinary(options: {
  url: string;
  method?: Method;
  data?: unknown;
  auth?: boolean;
}): Promise<ArrayBuffer> {
  return new Promise((resolve, reject) => {
    const headers: Record<string, string> = {};
    if (options.auth !== false) {
      const token = getToken();
      if (token) headers.Authorization = `Bearer ${token}`;
    }

    wx.request({
      url: `${baseUrl()}${options.url}`,
      method: (options.method || 'GET') as WechatMiniprogram.RequestOption['method'],
      data: options.data as WechatMiniprogram.RequestOption['data'],
      header: headers,
      responseType: 'arraybuffer',
      success(res) {
        if (res.statusCode === 401) {
          clearToken();
          wx.redirectTo({ url: '/pages/home/index' });
          reject(new Error('unauthorized'));
          return;
        }
        if (res.statusCode < 200 || res.statusCode >= 300 || !(res.data instanceof ArrayBuffer)) {
          reject(new Error('request failed'));
          return;
        }
        resolve(res.data);
      },
      fail: reject,
    });
  });
}
