"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.login = login;
const http_1 = require("./http");
const storage_1 = require("../utils/storage");
async function login() {
    const code = await new Promise((resolve, reject) => {
        wx.login({
            success: (res) => resolve(res.code),
            fail: reject,
        });
    });
    const result = await (0, http_1.request)({
        url: '/api/auth/wechat-login',
        method: 'POST',
        data: { code },
        auth: false,
    });
    (0, storage_1.setToken)(result.token);
}
//# sourceMappingURL=auth.service.js.map