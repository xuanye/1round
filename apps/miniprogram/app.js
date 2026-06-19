"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const config_1 = require("./utils/config");
App({
    onLaunch() {
        // Local subset Material Symbols font is loaded offline in app.wxss
    },
    onError(err) {
        console.error('Global application error caught:', err);
    },
    onPageNotFound(res) {
        console.error('Page not found:', res.path);
        wx.redirectTo({ url: '/pages/home/index' }); // use redirectTo since home is not a tabbar page in app.json
    },
    globalData: {
        baseUrl: (0, config_1.getBaseUrl)(),
    },
});
//# sourceMappingURL=app.js.map