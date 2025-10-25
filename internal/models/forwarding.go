package models

import (
	"time"
)

type ForwardingEntry struct {
	Name      string    `json:"name"`
	Target    string    `json:"target"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DomainEntry struct {
	Domain    string    `json:"domain"`
	Target    string    `json:"target"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DomainEntryPublic 公开的域名信息，不包含敏感token
type DomainEntryPublic struct {
	Domain    string    `json:"domain"`
	Target    string    `json:"target"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Response struct {
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
}

// BatchUpdateEntry 批量更新的单个条目
type BatchUpdateEntry struct {
	Name   string `json:"name,omitempty"`   // 路径重定向的名称
	Domain string `json:"domain,omitempty"` // 域名重定向的域名
	Target string `json:"target"`           // 目标地址
}

// BatchUpdateRequest 批量更新请求 (POST JSON body)
type BatchUpdateRequest struct {
	RedirectToken string             `json:"redirect_token,omitempty"` // 路径重定向的token
	DomainToken   string             `json:"domain_token,omitempty"`   // 域名重定向的token
	Entries       []BatchUpdateEntry `json:"entries"`
}

// BatchUpdateResponse 批量更新响应
type BatchUpdateResponse struct {
	State    string                    `json:"state"`
	Message  string                    `json:"message,omitempty"`
	Results  []BatchUpdateEntryResult  `json:"results,omitempty"`
	Summary  BatchUpdateSummary        `json:"summary"`
}

// BatchUpdateEntryResult 单个条目的更新结果
type BatchUpdateEntryResult struct {
	Name    string `json:"name,omitempty"`
	Domain  string `json:"domain,omitempty"`
	Target  string `json:"target"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// BatchUpdateSummary 批量更新汇总
type BatchUpdateSummary struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}