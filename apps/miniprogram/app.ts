App({
  onLaunch() {
    wx.loadFontFace({
      family: 'Material Symbols Outlined',
      source: 'url("https://fonts.gstatic.com/s/materialsymbolsoutlined/v218/y3HsZy1sJx-1pdcQG9O30sB8qaNqdqy2D3J3S7W9q01k.woff2")',
      scopes: ['native', 'webview'],
      success: () => console.log('Material Symbols font loaded successfully'),
      fail: (err) => console.error('Failed to load Material Symbols font', err)
    });
  },
  globalData: {
    baseUrl: 'http://localhost:8080',
  },
});
