package constants

type SaveLoginInfo struct {
	Token     string `json:"token"`
	ServerUrl string `json:"serverUrl"`
}

type RespStatus struct {
	Success bool `json:"success"`
}
