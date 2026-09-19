package n5

type Egress struct {
	Id                int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Name              string `json:"name" form:"name" gorm:"index"`
	Remark            string `json:"remark" form:"remark"`
	Protocol          string `json:"protocol" form:"protocol"`
	Tag               string `json:"tag" form:"tag" gorm:"uniqueIndex"`
	Enabled           bool   `json:"enabled" form:"enabled" gorm:"default:true"`
	OutboundJSON      string `json:"outboundJson" form:"outboundJson" gorm:"column:outbound_json;type:text"`
	LastStatus        string `json:"lastStatus" form:"lastStatus" gorm:"column:last_status;default:''"`
	LastExitIP        string `json:"lastExitIP" form:"lastExitIP" gorm:"column:last_exit_ip;default:''"`
	LastTestTime      int64  `json:"lastTestTime" form:"lastTestTime" gorm:"column:last_test_time;default:0"`
	LastTestStatus    string `json:"lastTestStatus" form:"lastTestStatus" gorm:"column:last_test_status;default:''"`
	LastTestLatencyMs int    `json:"lastTestLatencyMs" form:"lastTestLatencyMs" gorm:"column:last_test_latency_ms;default:0"`
	LastTestError     string `json:"lastTestError" form:"lastTestError" gorm:"column:last_test_error;type:text"`
	LastTestAt        int64  `json:"lastTestAt" form:"lastTestAt" gorm:"column:last_test_at;default:0"`
	CreatedAt         int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt         int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (Egress) TableName() string {
	return "n5_egresses"
}

type EgressTest struct {
	Id       int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	EgressId int    `json:"egressId" form:"egressId" gorm:"column:egress_id;index"`
	Status   string `json:"status" form:"status" gorm:"default:''"`
	Latency  int    `json:"latency" form:"latency" gorm:"default:0"`
	ExitIP   string `json:"exitIp" form:"exitIp" gorm:"column:exit_ip;default:''"`
	Message  string `json:"message" form:"message" gorm:"type:text"`
	TestedAt int64  `json:"testedAt" form:"testedAt" gorm:"column:tested_at;index"`
}

func (EgressTest) TableName() string {
	return "n5_egress_test"
}

type EgressLabel struct {
	Id        int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Name      string `json:"name" form:"name" gorm:"index"`
	Type      string `json:"type" form:"type" gorm:"index"`
	CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (EgressLabel) TableName() string {
	return "n5_egress_labels"
}

type EgressLabelRelation struct {
	Id        int   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	EgressId  int   `json:"egressId" form:"egressId" gorm:"column:egress_id;uniqueIndex:idx_n5_egress_label_relation"`
	LabelId   int   `json:"labelId" form:"labelId" gorm:"column:label_id;uniqueIndex:idx_n5_egress_label_relation"`
	CreatedAt int64 `json:"createdAt" gorm:"autoCreateTime:milli"`
}

func (EgressLabelRelation) TableName() string {
	return "n5_egress_label_relations"
}

type EgressPool struct {
	Id               int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Name             string `json:"name" form:"name" gorm:"index"`
	Remark           string `json:"remark" form:"remark"`
	Tag              string `json:"tag" form:"tag" gorm:"uniqueIndex"`
	Strategy         string `json:"strategy" form:"strategy" gorm:"default:'random'"`
	FallbackType     string `json:"fallbackType" form:"fallbackType" gorm:"column:fallback_type;default:''"`
	FallbackTargetId int    `json:"fallbackTargetId" form:"fallbackTargetId" gorm:"column:fallback_target_id;default:0"`
	Enabled          bool   `json:"enabled" form:"enabled" gorm:"default:true"`
	CreatedAt        int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt        int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (EgressPool) TableName() string {
	return "n5_egress_pools"
}

type EgressPoolMember struct {
	Id        int   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	PoolId    int   `json:"poolId" form:"poolId" gorm:"uniqueIndex:idx_n5_pool_member;index"`
	EgressId  int   `json:"egressId" form:"egressId" gorm:"uniqueIndex:idx_n5_pool_member;index"`
	Weight    int   `json:"weight" form:"weight" gorm:"default:1"`
	SortOrder int   `json:"sortOrder" form:"sortOrder" gorm:"column:sort_order;default:0"`
	Enabled   bool  `json:"enabled" form:"enabled" gorm:"default:true"`
	CreatedAt int64 `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt int64 `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (EgressPoolMember) TableName() string {
	return "n5_egress_pool_members"
}

type TrafficPolicy struct {
	Id                int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Name              string `json:"name" form:"name" gorm:"index"`
	Remark            string `json:"remark" form:"remark"`
	Enabled           bool   `json:"enabled" form:"enabled" gorm:"default:true"`
	DefaultTargetType string `json:"defaultTargetType" form:"defaultTargetType" gorm:"column:default_target_type;default:''"`
	DefaultTargetId   int    `json:"defaultTargetId" form:"defaultTargetId" gorm:"column:default_target_id;default:0"`
	CreatedAt         int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt         int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (TrafficPolicy) TableName() string {
	return "n5_traffic_policies"
}

type TrafficPolicyRule struct {
	Id         int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	PolicyId   int    `json:"policyId" form:"policyId" gorm:"index"`
	RuleType   string `json:"ruleType" form:"ruleType" gorm:"column:rule_type"`
	MatchMode  string `json:"matchMode" form:"matchMode" gorm:"column:match_mode"`
	MatchValue string `json:"matchValue" form:"matchValue" gorm:"column:match_value;type:text"`
	TargetType string `json:"targetType" form:"targetType" gorm:"column:target_type"`
	TargetId   int    `json:"targetId" form:"targetId" gorm:"column:target_id;default:0"`
	SortOrder  int    `json:"sortOrder" form:"sortOrder" gorm:"column:sort_order;default:0"`
	Enabled    bool   `json:"enabled" form:"enabled" gorm:"default:true"`
	CreatedAt  int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt  int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (TrafficPolicyRule) TableName() string {
	return "n5_traffic_policy_rules"
}

type TrafficPolicyBinding struct {
	Id        int   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	InboundId int   `json:"inboundId" form:"inboundId" gorm:"index"`
	PolicyId  int   `json:"policyId" form:"policyId" gorm:"index"`
	Enabled   bool  `json:"enabled" form:"enabled" gorm:"default:true"`
	CreatedAt int64 `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt int64 `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (TrafficPolicyBinding) TableName() string {
	return "n5_traffic_policy_bindings"
}

type XrayConfigHistory struct {
	Id                  int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Source              string `json:"source" form:"source" gorm:"default:''"`
	BaseConfigHash      string `json:"baseConfigHash" form:"baseConfigHash" gorm:"column:base_config_hash;index"`
	ExtensionConfigHash string `json:"extensionConfigHash" form:"extensionConfigHash" gorm:"column:extension_config_hash;index"`
	ConfigHash          string `json:"configHash" form:"configHash" gorm:"column:config_hash;index"`
	ConfigJSON          string `json:"configJson" form:"configJson" gorm:"column:config_json;type:text"`
	ApplyStatus         string `json:"applyStatus" form:"applyStatus" gorm:"column:apply_status;default:''"`
	ApplyError          string `json:"applyError" form:"applyError" gorm:"column:apply_error;type:text"`
	CreatedAt           int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
}

func (XrayConfigHistory) TableName() string {
	return "n5_xray_config_history"
}
