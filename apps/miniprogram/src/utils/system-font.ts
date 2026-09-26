export function getSystemFontClass(): string {
  try {
    const device = wx.getDeviceInfo();
    // Older HarmonyOS clients can report Android; use OS text, never the brand.
    if (/ohos|harmony|鸿蒙/i.test(`${device.platform} ${device.system}`)) return 'font-harmony';
    if (device.platform === 'ios' || /ios/i.test(device.system)) return 'font-ios';
    if (device.platform === 'android') return 'font-android';
  } catch {
    // Older base libraries retain the platform's default sans-serif fallback.
  }
  return 'font-system';
}
