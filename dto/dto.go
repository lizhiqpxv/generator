package dto

// *********************************************** 配置代码开始 ***********************************************
// 文件名生成规则： 使用下面👇的 struct 名称做为前缀，加上对应的功能描述，如： instance_api.go
// 此结构体为生成代码的根据，必须包含 ID 字段， 不支持 bool 类型
// parameter => 表示是否需要做为参数
// required => 表示是否为必须的参数
// time => 表示是否为时间字段

var (
	StructMap = map[string]interface{}{
		"Firmware": User{},
	}
)

// User 用户设置
type User struct {
	Id            int64 `json:"id"`
	Face          int   `json:"face" parameter:"true"`
	Fingerprint   int   `json:"fingerprint"parameter:"true"`
	Vibration     int   `json:"vibration" parameter:"true"`
	CutPower      int   `json:"cut_power" parameter:"true"`
	ChargeMonitor int   `json:"charge_monitor" parameter:"true"`
	GuardAlarm    int   `json:"guard_alarm" parameter:"true"`
	FaultAlarm    int   `json:"fault_alarm" parameter:"true"`
	CreatedAt     int64 `json:"created_at"`
	UpdatedAt     int64 `json:"updated_at"  parameter:"true"`
}
