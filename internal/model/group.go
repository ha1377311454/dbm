package model

// Group 连接分组
type Group struct {
	ID       string `json:"id"`
	Name     string `json:"name"`     // 分组名称
	ParentID string `json:"parentId"` // 父分组 ID (空字符串表示根分组)
}
