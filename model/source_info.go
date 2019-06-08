package model

// SourceInfoDetail ...
type SourceInfoDetail struct {
	SourceID        string   `xorm:"-" json:"id"`
	PublicKey       string   `json:"public_key"`
	Addresses       []string `xorm:"json" json:"addresses"` //一组节点源列表
	AgentVersion    string   `json:"agent_version"`
	ProtocolVersion string   `json:"protocol_version"`
}

// SourceInfo ...
type SourceInfo struct {
	Model             `xorm:"extends"`
	*SourceInfoDetail `xorm:"extends"`
}

func init() {
	RegisterTable(SourceInfo{})
}
