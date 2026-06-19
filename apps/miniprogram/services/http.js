"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.request = request;
const storage_1 = require("../utils/storage");
function baseUrl() {
    const app = getApp();
    return app.globalData.baseUrl;
}
async function request(options) {
    return new Promise((resolve, reject) => {
        const headers = { 'Content-Type': 'application/json' };
        if (options.auth !== false) {
            const token = (0, storage_1.getToken)();
            if (token)
                headers.Authorization = `Bearer ${token}`;
        }
        wx.request({
            url: `${baseUrl()}${options.url}`,
            method: (options.method || 'GET'),
            data: options.data,
            header: headers,
            success(res) {
                var _a;
                if (res.statusCode === 401) {
                    (0, storage_1.clearToken)();
                    wx.redirectTo({ url: '/pages/home/index' });
                    reject(new Error('unauthorized'));
                    return;
                }
                if (!res.data || res.data.code !== 0 || res.data.data === null) {
                    reject(new Error(((_a = res.data) === null || _a === void 0 ? void 0 : _a.message) || 'request failed'));
                    return;
                }
                resolve(res.data.data);
            },
            fail: reject,
        });
    });
}
//# sourceMappingURL=http.js.map