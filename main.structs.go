package main

const (
	version       = "1.3.1"
	timeLayout    = "2006-01-02T15:04:05.000Z"
	cliTimeLayout = "2006/01/02 15:04:05"
)

var (
	configFileName string
	configDebug    bool
	configDryRun   bool
	configVersion  bool
	logEntryType   = 3
	apiCallConfig  apiCallStruct
	timeNow        string
	dowMap         = make(map[string]int)
)

//----- Config Data Structs
type apiCallStruct struct {
	APIKey     string
	InstanceID string
	Zone       string
	Schedule   []apiSchedStruct
}

type apiSchedStruct struct {
	Enabled                bool
	CronSchedule           string
	DayOfMonthANDDayOfWeek bool
	ScheduleFrom           string
	ScheduleTo             string
	Service                string
	API                    string
	APIParams              map[string]apiParamStruct
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
