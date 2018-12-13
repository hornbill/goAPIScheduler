package main

const (
	version    = "1.1.0"
	timeLayout = "2006-01-02T15:04:05.000Z"
)

var (
	configFileName string
	configDryRun   bool
	logEntryType   = 3
	apiCallConfig  apiCallStruct
	boolConfLoaded bool
	timeNow        string
)

//----- Config Data Structs
type apiCallStruct struct {
	APIKey     string
	InstanceID string
	Zone       string
	Schedule   []apiSchedStruct
}

type apiSchedStruct struct {
	Enabled      bool
	CronSchedule string
	ScheduleFrom string
	ScheduleTo   string
	Service      string
	API          string
	APIParams    map[string]apiParamStruct
}

type apiParamStruct struct {
	Type      string
	Parameter string
	Content   string
}

type xmlmcResponse struct {
	MethodResult string      `xml:"status,attr"`
	State        stateStruct `xml:"state"`
}

type stateStruct struct {
	Code     string `xml:"code"`
	ErrorRet string `xml:"error"`
}