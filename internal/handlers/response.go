package handlers

import "betest/internal/response"

// Re-export response functions for convenience
var (
	SendSuccess     = response.SendSuccess
	SendError       = response.SendError
	SendSuccessNoData = response.SendSuccessNoData
)
