package controller

import (
	"fmt"
	"net/http"
	"one-api/common/config"
	"one-api/common/notify"
	"one-api/model"
	"one-api/types"
	"strings"

	"github.com/gin-gonic/gin"
)

func shouldEnableChannel(err error, openAIErr *types.OpenAIErrorWithStatusCode) bool {
	if !config.AutomaticEnableChannelEnabled {
		return false
	}
	if err != nil {
		return false
	}
	if openAIErr != nil {
		return false
	}
	return true
}

func ShouldDisableChannel(channelType int, err *types.OpenAIErrorWithStatusCode) bool {
	if !config.AutomaticDisableChannelEnabled {
		return false
	}

	if err == nil {
		return false
	}

	if err.LocalError {
		return false
	}

	if err.StatusCode == http.StatusUnauthorized {
		return true
	}

	if err.StatusCode == http.StatusForbidden {
		switch channelType {
		case config.ChannelTypeGemini:
			return true
		}
	}

	switch err.OpenAIError.Code {
	case "invalid_api_key":
		return true
	case "account_deactivated":
		return true
	case "billing_not_active":
		return true
	}

	switch err.Type {
	case "insufficient_quota":
		return true
	// https://docs.anthropic.com/claude/reference/errors
	case "authentication_error":
		return true
	case "permission_error":
		return true
	case "forbidden":
		return true
	}

	if strings.Contains(err.OpenAIError.Message, "Your credit balance is too low") { // anthropic
		return true
	} else if strings.Contains(err.OpenAIError.Message, "This organization has been disabled.") {
		return true
	} else if strings.Contains(err.OpenAIError.Message, "You exceeded your current quota") {
		return true
	} else if strings.Contains(err.OpenAIError.Message, "Permission denied") {
		return true
	}

	if strings.Contains(err.OpenAIError.Message, "credit") {
		return true
	}
	if strings.Contains(err.OpenAIError.Message, "balance") {
		return true
	}

	if strings.Contains(err.OpenAIError.Message, "Access denied") {
		return true
	}
	return false

}

// disable & notify
func DisableChannel(channelId int, channelName string, reason string, sendNotify bool) {
	model.UpdateChannelStatusById(channelId, config.ChannelStatusAutoDisabled)
	if !sendNotify {
		return
	}

	subject := fmt.Sprintf("通道「%s」（#%d）已被禁用", channelName, channelId)
	content := fmt.Sprintf("通道「%s」（#%d）已被禁用，原因：%s", channelName, channelId, reason)
	notify.Send(subject, content)
}

// enable & notify
func EnableChannel(channelId int, channelName string, sendNotify bool) {
	model.UpdateChannelStatusById(channelId, config.ChannelStatusEnabled)
	if !sendNotify {
		return
	}

	subject := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
	content := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
	notify.Send(subject, content)
}

func RelayNotFound(c *gin.Context) {
	err := types.OpenAIError{
		Message: fmt.Sprintf("Invalid URL (%s %s)", c.Request.Method, c.Request.URL.Path),
		Type:    "invalid_request_error",
		Param:   "",
		Code:    "",
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": err,
	})
}
