package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/hornbill/color"
	"github.com/hornbill/cron"
	"github.com/hornbill/goApiLib"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

const (
	version    = "1.0.0"
	timeLayout = "2006-01-02T15:04:05.000Z"
)

var (
	configFileName      string
	apiCallConfig       apiCallStruct
	boolConfLoaded      bool
	timeNow             string
	xmlmcInstanceConfig xmlmcConfigStruct
)

//----- Config Data Structs
type apiCallStruct struct {
	APIKey     string
	InstanceID string
	Zone       string
	Schedule   []apiSchedStruct
	URL        string
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

//----- XMLMC Config and Interaction Structs
type xmlmcConfigStruct struct {
	instance string
	url      string
	zone     string
}

type xmlmcResponse struct {
	MethodResult string      `xml:"status,attr"`
	State        stateStruct `xml:"state"`
}

type stateStruct struct {
	Code     string `xml:"code"`
	ErrorRet string `xml:"error"`
}

func main() {
	//-- Start Time for Log File
	timeNow = time.Now().Format(time.RFC3339)
	timeNow = strings.Replace(timeNow, ":", "-", -1)

	//Time for CLI output
	currTime := time.Now().Format(time.RFC3339)
	currTime = strings.Replace(currTime, "-", "/", -1)
	currTime = strings.Replace(currTime, "T", " ", 1)
	currTime = strings.Replace(currTime, "Z", "", 1)

	flag.StringVar(&configFileName, "file", "conf.json", "Name of the configuration file to load")

	logger(3, "---- Hornbill API Scheduler V"+fmt.Sprintf("%v", version)+" ----", true)
	logger(3, "Flag - Configuration File: "+fmt.Sprintf("%s", configFileName), true)
	logger(3, "Scheduler started at: "+currTime, true)

	//-- Load Configuration File Into Struct
	apiCallConfig, boolConfLoaded = loadConfig()
	if boolConfLoaded != true {
		logger(4, "Unable to load config, process closing.", true)
		return
	}

	//-- Set Instance ID
	SetInstance(apiCallConfig.Zone, apiCallConfig.InstanceID)
	//-- Generate Instance XMLMC Endpoint
	apiCallConfig.URL = getInstanceURL()

	logger(3, "Enabled Schedule Items:", true)
	c := cron.New()
	c.Start()
	scheduleCount := 0
	for _, scheduleEntry := range apiCallConfig.Schedule {
		scheduleEntryUnique := scheduleEntry
		if scheduleEntryUnique.Enabled == true {
			scheduleCount++
			logger(3, "----====----", true)
			logger(3, "Cron Schedule: "+scheduleEntryUnique.CronSchedule, true)
			logger(3, "Service: "+scheduleEntryUnique.Service, true)
			logger(3, "API: "+scheduleEntryUnique.API, true)
			c.AddFunc(scheduleEntryUnique.CronSchedule, func() { apiRequest(scheduleEntryUnique) })
		}
	}

	if scheduleCount == 0 {
		logger(3, "There are no scheduled tasks enabled!", true)
		return
	}
	logger(3, "----====----", true)

	//Leave running, wait for user interruption
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)
	<-sig

}

func apiRequest(scheduleEntry apiSchedStruct) {
	//-- Time for CLI output
	currTime := time.Now().Format(time.RFC3339)
	currTime = strings.Replace(currTime, "-", "/", -1)
	currTime = strings.Replace(currTime, "T", " ", 1)
	currTime = strings.Replace(currTime, "Z", "", 1)

	espXmlmc := apiLib.NewXmlmcInstance(apiCallConfig.URL)
	espXmlmc.SetAPIKey(apiCallConfig.APIKey)

	timeNow := time.Now()
	timeFrom := timeNow
	timeTo := timeNow

	if scheduleEntry.ScheduleFrom != "" || scheduleEntry.ScheduleTo != "" {
		if scheduleEntry.ScheduleFrom != "" {
			timeFromParsed, err := time.Parse(timeLayout, scheduleEntry.ScheduleFrom)
			if err != nil {
				color.Red("\n[ERROR] " + currTime + " ScheduleFrom date.time parse failed, unable to run API [" + scheduleEntry.Service + "::" + scheduleEntry.API + "]")
				logger(4, "Unable to parse time string in ScheduleFrom for request ["+scheduleEntry.Service+"::"+scheduleEntry.API+"] : "+fmt.Sprintf("%v", err), false)
				return
			}
			timeFrom = timeFromParsed
		}
		if scheduleEntry.ScheduleTo != "" {
			timeToParsed, err := time.Parse(timeLayout, scheduleEntry.ScheduleTo)
			if err != nil {
				color.Red("\n[ERROR] " + currTime + " ScheduleTo date.time parse failed, unable to run API [" + scheduleEntry.Service + "::" + scheduleEntry.API + "]")
				logger(4, "Unable to parse time string in ScheduleTo for request ["+scheduleEntry.Service+"::"+scheduleEntry.API+"] : "+fmt.Sprintf("%v", err), false)
				return
			}
			timeTo = timeToParsed
		}
	}
	if timeTo.Before(timeNow) {
		logger(1, "Current time is after the schedule entry is set to end: "+scheduleEntry.Service+"::"+scheduleEntry.API, true)
		return
	}
	if timeFrom.After(timeNow) {
		logger(1, "Current time is before the schedule entry is set to start: "+scheduleEntry.Service+"::"+scheduleEntry.API, true)
		return
	}

	//Build params
	for i := 0; i < len(scheduleEntry.APIParams); i++ {
		switch scheduleEntry.APIParams[strconv.Itoa(i)].Type {
		case "Content":
			espXmlmc.SetParam(scheduleEntry.APIParams[strconv.Itoa(i)].Parameter, scheduleEntry.APIParams[strconv.Itoa(i)].Content)
		case "Open":
			//Open XML element for complex params
			espXmlmc.OpenElement(scheduleEntry.APIParams[strconv.Itoa(i)].Parameter)
		case "Close":
			//Close XML element for complex params
			espXmlmc.CloseElement(scheduleEntry.APIParams[strconv.Itoa(i)].Parameter)
		}
	}

	var XMLSTRING = espXmlmc.GetParam()
	XMLAPICall, xmlmcErr := espXmlmc.Invoke(scheduleEntry.Service, scheduleEntry.API)
	if xmlmcErr != nil {
		color.Red("\n" + currTime + " API Call Failed: " + fmt.Sprintf("%v", xmlmcErr))
		logger(4, "API Call Failed: "+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "Request Log XML "+fmt.Sprintf("%s", XMLSTRING), false)
	} else {
		var xmlRespon xmlmcResponse
		err := xml.Unmarshal([]byte(XMLAPICall), &xmlRespon)
		if err != nil {
			color.Red("\n" + currTime + " API Call Failed: " + fmt.Sprintf("%v", err))
			logger(4, "API Call Failed: "+fmt.Sprintf("%v", err), false)
			logger(1, "Request Log XML "+fmt.Sprintf("%s", XMLSTRING), false)
		} else {
			if xmlRespon.MethodResult != "ok" {
				color.Red("\n" + currTime + " API Call Failed: " + xmlRespon.State.ErrorRet)
				logger(4, "API Call Failed: "+xmlRespon.State.ErrorRet, false)
				logger(1, "Request Log XML "+fmt.Sprintf("%s", XMLSTRING), false)
			} else {
				color.Green("\n" + currTime + " API Call Success: " + scheduleEntry.Service + "::" + scheduleEntry.API)
				logger(1, "API Call Success: "+scheduleEntry.Service+"::"+scheduleEntry.API, false)

			}
		}
	}
}

//loadConfig -- Function to Load Configruation File
func loadConfig() (apiCallStruct, bool) {
	boolLoadConf := true
	//-- Check Config File File Exists
	cwd, _ := os.Getwd()
	configurationFilePath := cwd + "/" + configFileName
	logger(1, "Loading Config File: "+configurationFilePath, false)
	if _, fileCheckErr := os.Stat(configurationFilePath); os.IsNotExist(fileCheckErr) {
		logger(4, "No Configuration File", true)
		os.Exit(102)
	}
	//-- Load Config File
	file, fileError := os.Open(configurationFilePath)
	//-- Check For Error Reading File
	if fileError != nil {
		logger(4, "Error Opening Configuration File: "+fmt.Sprintf("%v", fileError), true)
		boolLoadConf = false
	}

	//-- New Decoder
	decoder := json.NewDecoder(file)
	//-- New Var based on apiCallStruct
	edbConf := apiCallStruct{}
	//-- Decode JSON
	err := decoder.Decode(&edbConf)
	//-- Error Checking
	if err != nil {
		logger(4, "Error Decoding Configuration File: "+fmt.Sprintf("%v", err), true)
		boolLoadConf = false
	}
	//-- Return New Config
	return edbConf, boolLoadConf
}

// logger -- function to append to the current log file
func logger(t int, s string, outputtoCLI bool) {
	cwd, _ := os.Getwd()
	logPath := cwd + "/log"
	logFileName := logPath + "/HB_API_Scheduler_" + timeNow + ".log"

	//-- If Folder Does Not Exist then create it
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err := os.Mkdir(logPath, 0777)
		if err != nil {
			color.Red("Error Creating Log Folder %q: %s \r", logPath, err)
			os.Exit(101)
		}
	}

	//-- Open Log File
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	// don't forget to close it
	defer f.Close()
	if err != nil {
		color.Red("Error Creating Log File %q: %s \n", logFileName, err)
		os.Exit(100)
	}
	// assign it to the standard logger
	log.SetOutput(f)
	var errorLogPrefix string
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 2:
		errorLogPrefix = "[MESSAGE] "
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 3:
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 4:
		errorLogPrefix = "[ERROR] "
		if outputtoCLI {
			color.Set(color.FgRed)
			defer color.Unset()
		}
	case 5:
		errorLogPrefix = "[WARNING]"
		if outputtoCLI {
			color.Set(color.FgYellow)
			defer color.Unset()
		}
	case 6:
		if outputtoCLI {
			color.Set(color.FgYellow)
			defer color.Unset()
		}
	}

	if outputtoCLI {
		fmt.Printf("%v \n", errorLogPrefix+s)
	}

	log.Println(errorLogPrefix + s)
}

// SetInstance sets the Zone and Instance config from the passed-through strZone and instanceID values
func SetInstance(strZone string, instanceID string) {
	//-- Set Zone
	SetZone(strZone)
	//-- Set Instance
	xmlmcInstanceConfig.instance = instanceID
	return
}

// SetZone - sets the Instance Zone to Overide current live zone
func SetZone(zone string) {
	xmlmcInstanceConfig.zone = zone
	return
}

// getInstanceURL -- Function to build XMLMC End Point
func getInstanceURL() string {
	xmlmcInstanceConfig.url = "https://"
	xmlmcInstanceConfig.url += xmlmcInstanceConfig.zone
	xmlmcInstanceConfig.url += "api.hornbill.com/"
	xmlmcInstanceConfig.url += xmlmcInstanceConfig.instance
	xmlmcInstanceConfig.url += "/xmlmc/"
	return xmlmcInstanceConfig.url
}
