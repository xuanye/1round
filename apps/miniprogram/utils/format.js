"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.formatScore = formatScore;
exports.toInteger = toInteger;
function formatScore(score) {
    return score > 0 ? `+${score}` : `${score}`;
}
function toInteger(value) {
    if (!/^-?\d+$/.test(value.trim()))
        return null;
    return Number(value);
}
//# sourceMappingURL=format.js.map