App({
  onLaunch() {
    wx.loadFontFace({
      family: 'Material Symbols Outlined',
      source: 'url("https://fonts.gstatic.com/s/materialsymbolsoutlined/v351/kJF1BvYX7BgnkSrUwT8OhrdQw4oELdPIeeII9v6oDMzByHX9rA6RzaxHMPdY43zj-jCxv3fzvRNU22ZXGJpEpjC_1v-p_4MrImHCIJIZrD_xHOembdhzrA.ttf")',
      scopes: ['native', 'webview'],
      global: true,
      success: () => console.log('Material Symbols font loaded successfully'),
      fail: (err) => console.error('Failed to load Material Symbols font', err)
    });
  },
  globalData: {
    baseUrl: 'http://localhost:8080',
  },
});
