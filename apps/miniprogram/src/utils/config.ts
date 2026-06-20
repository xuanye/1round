const DEFAULT_BASE_URL = 'https://1round.xuanye.wang';
const BUILD_API_BASE_URL = '__ONEROUND_API_BASE_URL__';
const BUILD_API_BASE_URL_TOKEN = '__' + 'ONEROUND_API_BASE_URL' + '__';

export function getBaseUrl(): string {
  if (BUILD_API_BASE_URL && BUILD_API_BASE_URL !== BUILD_API_BASE_URL_TOKEN) {
    return BUILD_API_BASE_URL;
  }

  return DEFAULT_BASE_URL;
}
