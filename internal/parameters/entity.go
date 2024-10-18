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
	DEV_PID string `json:"DEV_PID"`
}
