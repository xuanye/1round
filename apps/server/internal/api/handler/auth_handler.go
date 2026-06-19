package handler

import (
	"encoding/json"
	"net/http"

	"github.com/xuanye/one-round/apps/server/internal/api/dto"
	"github.com/xuanye/one-round/apps/server/internal/api/response"
	authsvc "github.com/xuanye/one-round/apps/server/internal/app/auth"
)

type AuthHandler struct {
	auth *authsvc.Service
}

func NewAuthHandler(auth *authsvc.Service) *AuthHandler { return &AuthHandler{auth: auth} }

func (h *AuthHandler) WechatLogin(w http.ResponseWriter, r *http.Request) {
	var req dto.WechatLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	result, err := h.auth.LoginWithWechatCode(r.Context(), req.Code)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
