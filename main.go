package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-xmlfmt/xmlfmt"
	"github.com/hornbill/color"
	apiLib "github.com/hornbill/goApiLib"
	"github.com/robfig/cron"
)

func main() {
	//-- Start Time for Log File Name
	timeNow = time.Now().Format("20060102150405")
	//Time for CLI output & Log File Content
	currTime := time.Now().Format(cliTimeLayout)

	flag.StringVar(&configFileName, "file", "conf.json", "Name of the configuration file to load")
	flag.BoolVar(&configDebug, "debug", false, "Set to true to run scheduler in debug mode, where API call request and reponse XML payload will be written to the log")
	flag.BoolVar(&configDryRun, "dryrun", false, "Set to true to run scheduler in dryrun mode. No API calls will be made")
	flag.BoolVar(&configVersion, "version", false, "Set to true to output the version")
	flag.Parse()
	if configDryRun {
		logEntryType = 6
		logger(logEntryType, "RUNNING IN DRYRUN MODE", true)
	}
	//-- If configVersion just output version number and die
	if configVersion {
		fmt.Printf("%v \n", version)
		return
	}
	logger(logEntryType, "---- Hornbill API Scheduler V"+fmt.Sprintf("%v", version)+" ----", true)
	logger(logEntryType, "Flag - Configuration File: "+configFileName, true)
	logger(logEntryType, "Scheduler started at: "+currTime, true)

	//-- Load Configuration File Into Struct
	apiCallConfig = loadConfig()

	logger(logEntryType, "Enabled Schedule Items:", true)
	c := cron.New()
	c.Start()
	scheduleCount := 0
	for _, scheduleEntry := range apiCallConfig.Schedule {
		scheduleEntryUnique := scheduleEntry
		if scheduleEntryUnique.Enabled {
			scheduleCount++
			logger(logEntryType, "----====----", true)
			logger(logEntryType, "Cron Schedule: "+scheduleEntryUnique.CronSchedule, true)
			logger(logEntryType, "Service: "+scheduleEntryUnique.Service, true)
			logger(logEntryType, "API: "+scheduleEntryUnique.API, true)
			c.AddFunc(scheduleEntryUnique.CronSchedule, func() { apiRequest(scheduleEntryUnique) })
		}
	}

	if scheduleCount == 0 {
		logger(logEntryType, "There are no scheduled tasks enabled!", true)
		return
	}
	logger(logEntryType, "----====----", true)

	//Leave running, wait for user interruption
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)
	<-sig

}

func apiRequest(scheduleEntry apiSchedStruct) {
	espXmlmc := apiLib.NewXmlmcInstance(apiCallConfig.InstanceID)
	espXmlmc.SetAPIKey(apiCallConfig.APIKey)

	timeNow := time.Now()
	timeFrom := timeNow
	timeTo := timeNow

	if scheduleEntry.ScheduleFrom != "" || scheduleEntry.ScheduleTo != "" {
		if scheduleEntry.ScheduleFrom != "" {
			timeFromParsed, err := time.Parse(timeLayout, scheduleEntry.ScheduleFrom)
			if err != nil {
				color.Red("\n[ERROR] " + time.Now().Format(cliTimeLayout) + " ScheduleFrom date.time parse failed, unable to run API [" + scheduleEntry.Service + "::" + scheduleEntry.API + "]")
				logger(4, "Unable to parse time string in ScheduleFrom for request ["+scheduleEntry.Service+"::"+scheduleEntry.API+"] : "+fmt.Sprintf("%v", err), false)
				return
			}
			timeFrom = timeFromParsed
		}
		if scheduleEntry.ScheduleTo != "" {
			timeToParsed, err := time.Parse(timeLayout, scheduleEntry.ScheduleTo)
			if err != nil {
				color.Red("\n[ERROR] " + time.Now().Format(cliTimeLayout) + " ScheduleTo date.time parse failed, unable to run API [" + scheduleEntry.Service + "::" + scheduleEntry.API + "]")
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
			var strContent = scheduleEntry.APIParams[strconv.Itoa(i)].Content

			//Check to see if param content contains nowPlus::int::
			var timeAddRegex = regexp.MustCompile(`nowPlus::[0-9]{0,5}::`)
			var boolDynDate = timeAddRegex.MatchString(strContent)

			//If we have a dynamic date requirement in the config, process
			if boolDynDate {
				result := strings.Split(strContent, "::")
				//Setup time object
				var timeThen = time.Now()
				//Get vals from array
				var timeAddValue, _ = strconv.Atoi(result[1])
				var timeAddUnit = strings.ToLower(result[2])

				if timeAddUnit == "years" {
					timeThen = timeThen.AddDate(timeAddValue, 0, 0)
					strContent = timeThen.Format("2006-01-02 15:04:05")
				} else if timeAddUnit == "months" {
					timeThen = timeThen.AddDate(0, timeAddValue, 0)
					strContent = timeThen.Format("2006-01-02 15:04:05")
				} else if timeAddUnit == "days" {
					timeThen = timeThen.AddDate(0, 0, timeAddValue)
					strContent = timeThen.Format("2006-01-02 15:04:05")
				} else if timeAddUnit == "hours" {
					timeThen = timeThen.Add(time.Duration(timeAddValue) * time.Hour)
					strContent = timeThen.Format("2006-01-02 15:04:05")
				} else if timeAddUnit == "minutes" {
					timeThen = timeThen.Add(time.Duration(timeAddValue) * time.Minute)
					strContent = timeThen.Format("2006-01-02 15:04:05")
				}
			}

			espXmlmc.SetParam(scheduleEntry.APIParams[strconv.Itoa(i)].Parameter, strContent)
		case "Open":
			//Open XML element for complex params
			espXmlmc.OpenElement(scheduleEntry.APIParams[strconv.Itoa(i)].Parameter)
		case "Close":
			//Close XML element for complex params
			espXmlmc.CloseElement(scheduleEntry.APIParams[strconv.Itoa(i)].Parameter)
		}
	}

	var XMLSTRING = espXmlmc.GetParam()
	if configDryRun {
		color.Yellow(time.Now().Format(cliTimeLayout) + " DRYRUN API Call details have been added to the log")
		logger(logEntryType, "----DRYRUN API Request----", false)
		logger(logEntryType, "Service: "+scheduleEntry.Service, false)
		logger(logEntryType, "Method: "+scheduleEntry.API, false)
		logger(logEntryType, "XML Parameters: "+XMLSTRING, false)
		return
	}

	color.Green("\n" + time.Now().Format(cliTimeLayout) + " Executing API Call: " + scheduleEntry.Service + "::" + scheduleEntry.API)
	logger(logEntryType, "Executing API Call: "+scheduleEntry.Service+"::"+scheduleEntry.API, false)
	if configDebug {
		logger(1, "API Request Payload:\n"+xmlfmt.FormatXML(XMLSTRING, "\t", "	"), false)
	}

	XMLAPICall, xmlmcErr := espXmlmc.Invoke(scheduleEntry.Service, scheduleEntry.API)
	if xmlmcErr != nil {
		color.Red(time.Now().Format(cliTimeLayout) + " API Call Failed: " + fmt.Sprintf("%v", xmlmcErr))
		logger(4, "API Call Failed: "+fmt.Sprintf("%v", xmlmcErr), false)
	} else {
		if configDebug {
			logger(1, "API Response Payload:\n"+xmlfmt.FormatXML(XMLAPICall, "\t", "	"), false)
		}

		var xmlRespon xmlmcResponse
		err := xml.Unmarshal([]byte(XMLAPICall), &xmlRespon)
		if err != nil {
			color.Red(time.Now().Format(cliTimeLayout) + " API Call Failed: " + fmt.Sprintf("%v", err))
			logger(4, "API Call Failed: "+fmt.Sprintf("%v", err), false)
		} else {
			if xmlRespon.MethodResult != "ok" {
				color.Red(time.Now().Format(cliTimeLayout) + " API Call Failed: " + xmlRespon.State.ErrorRet)
				logger(4, "API Call Failed: "+xmlRespon.State.ErrorRet, false)
			} else {
				color.Green(time.Now().Format(cliTimeLayout) + " API Call Success: " + scheduleEntry.Service + "::" + scheduleEntry.API)
				logger(logEntryType, "API Call Success: "+scheduleEntry.Service+"::"+scheduleEntry.API, false)
			}
		}
	}
}

//loadConfig -- Function to Load Configruation File
func loadConfig() apiCallStruct {
	//-- Check Config File File Exists
	cwd, _ := os.Getwd()
	configurationFilePath := cwd + "/" + configFileName
	logger(logEntryType, "Loading Config File: "+configurationFilePath, false)
	if _, fileCheckErr := os.Stat(configurationFilePath); os.IsNotExist(fileCheckErr) {
		logger(4, "No Configuration File", true)
		os.Exit(102)
	}
	//-- Load Config File
	file, fileError := os.Open(configurationFilePath)
	//-- Check For Error Reading File
	if fileError != nil {
		logger(4, "Error Opening Configuration File: "+fmt.Sprintf("%v", fileError), true)
		os.Exit(102)
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
		os.Exit(102)
	}
	//-- Return New Config
	return edbConf
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
	if err != nil {
		color.Red("Error Creating Log File %q: %s \n", logFileName, err)
		os.Exit(100)
	}
	defer f.Close()
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
