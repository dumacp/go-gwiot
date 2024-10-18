package parameters

type PlatformParameters struct {
	ID                  string   `json:"id"`
	ApplicationIds      []string `json:"applicationIds"`
	GroupID             string   `json:"groupId"`
	HasErrors           bool     `json:"hasErrors"`
	NewParameterMapHash string   `json:"newParameterMapHash"`
	ParameterMapId      string   `json:"parameterMapId"`
	Props               *Props   `json:"props"`
	Timestamp           int64    `json:"timestamp"`
	VehicleId           string   `json:"vehicleId"`
}

type Props struct {
	DEV_PID               string `json:"DEV_PID"`
	BROKER_TOPIC_EXTERNAL string `json:"BROKER_TOPIC_EXTERNAL"`
	BROKER_URL_EXTERNAL   string `json:"BROKER_URL_EXTERNAL"`
	BROKER_USER_EXTERNAL  string `json:"BROKER_USER_EXTERNAL"`
	BROKER_PASS_EXTERNAL  string `json:"BROKER_PASS_EXTERNAL"`
}
