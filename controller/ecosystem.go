package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type EcosystemTokenUpsertRequest = service.EcosystemTokenUpsertRequest
type EcosystemTokenUpsertResponse = service.EcosystemTokenUpsertResponse
type EcosystemUserInfo = service.EcosystemUserInfo

func ecosystemUserFromContext(c *gin.Context) (*model.User, bool, error) {
	info, ok := c.Get(middleware.LogtoContextInfo)
	if !ok {
		return nil, false, service.ErrEcosystemInvalidToken
	}
	tokenInfo, ok := info.(*middleware.LogtoTokenInfo)
	if !ok {
		return nil, false, service.ErrEcosystemInvalidToken
	}
	user, created, err := service.ResolveOrProvisionUser(tokenInfo.Subject, tokenInfo.Email, "", "")
	if err != nil {
		return nil, false, err
	}
	if user.Status != common.UserStatusEnabled {
		return nil, false, service.ErrEcosystemUserDisabled
	}
	return user, created, nil
}

func EcosystemMe(c *gin.Context) {
	info, ok := c.Get(middleware.LogtoContextInfo)
	if !ok {
		common.ApiErrorMsg(c, service.ErrEcosystemInvalidToken.Error())
		return
	}
	tokenInfo, ok := info.(*middleware.LogtoTokenInfo)
	if !ok {
		common.ApiErrorMsg(c, service.ErrEcosystemInvalidToken.Error())
		return
	}
	user, created, err := service.ResolveOrProvisionUser(tokenInfo.Subject, tokenInfo.Email, "", "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, EcosystemUserInfo{
		LogtoSub:     tokenInfo.Subject,
		NewAPIUserID: user.Id,
		Username:     user.Username,
		Email:        user.Email,
		Group:        user.Group,
		Provisioned:  created,
	})
}

func EcosystemGroups(c *gin.Context) {
	user, _, err := ecosystemUserFromContext(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, service.GetEcosystemUserGroups(user))
}

func EcosystemModels(c *gin.Context) {
	user, _, err := ecosystemUserFromContext(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, service.GetEcosystemUserModels(user))
}

func EcosystemTokenUpsert(c *gin.Context) {
	user, _, err := ecosystemUserFromContext(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req EcosystemTokenUpsertRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	req.AppID = strings.TrimSpace(req.AppID)
	req.Capability = strings.TrimSpace(req.Capability)
	req.Group = strings.TrimSpace(req.Group)
	if req.Capability == "" {
		req.Capability = "default"
	}
	token, apiKey, created, err := service.UpsertEcosystemToken(user, req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	baseURL := service.GetLogtoEcosystemBaseURL()
	common.ApiSuccess(c, EcosystemTokenUpsertResponse{
		TokenID:    token.Id,
		TokenName:  token.Name,
		APIKey:     apiKey,
		BaseURL:    baseURL,
		Group:      token.Group,
		Capability: req.Capability,
		Created:    created,
	})
}
