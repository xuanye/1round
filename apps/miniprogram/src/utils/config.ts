export function getBaseUrl(): string {
  try {
    const accountInfo = wx.getAccountInfoSync();
    const env = accountInfo.miniProgram.envVersion;
    if (env === 'develop') {
      return 'http://localhost:8080';
    } else if (env === 'trial') {
      return 'https://api-staging.example.com';
    } else if (env === 'release') {
      return 'https://api.example.com';
    }
  } catch (e) {
    // fallback
  }
  return 'http://localhost:8080';
}
