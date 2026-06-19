"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.login = login;
exports.ensureLogin = ensureLogin;
const http_1 = require("./http");
const storage_1 = require("../utils/storage");
let loginPromise = null;
async function login() {
    if (loginPromise)
        return loginPromise;
    loginPromise = (async () => {
        try {
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
            (0, storage_1.setUser)({
                id: result.user.id,
                displayName: result.user.displayName,
                avatarUrl: result.user.avatarUrl,
            });
        }
        finally {
            loginPromise = null;
        }
    })();
    return loginPromise;
}
async function ensureLogin() {
    const token = (0, storage_1.getToken)();
    if (token)
        return;
    await login();
}
//# sourceMappingURL=auth.service.js.map