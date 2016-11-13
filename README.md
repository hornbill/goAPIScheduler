# Hornbill API Scheduler

The utility provides a quick and easy way to schedule the running of Hornbill API's.

## Quick Links
- [Installation](#installation)
- [Configuration](#configuration)
- [Execute](#execute)
- [Exit](#exit)

## Installation

#### Windows
* Download the ZIP archive containing the executables, configuration file and license;
* Extract the ZIP archive into a folder you would like the application to run from e.g. 'C:\hornbill_scheduler\'.

## Configuration

Example JSON File:

```
{
  "APIKey": "apikeygoeshere",
  "InstanceID": "instanceIDgoeshere",
  "Zone":"eur",
  "Schedule": [
    {
      "Enabled":true,
      "CronSchedule":"0 1 23 * * 1-6",
      "ScheduleFrom":"2016-11-12T00:00:00.000Z",
      "ScheduleTo":"2017-01-01T00:00:00.000Z",
      "Service":"apps/com.hornbill.servicemanager/Incidents",
      "API":"logIncident",
      "APIParams":{
        "0":
        {
          "Type":"Content",
          "Parameter":"summary",
          "Content":"Request Summary"
        },
        "1":
        {
          "Type":"Content",
          "Parameter":"description",
          "Content":"Request Description"
        },
        "2":
        {
          "Type":"Content",
          "Parameter":"customerId",
          "Content":"alanc"
        }
      }
    },
    {
      "Enabled":true,
      "CronSchedule":"0 25 19 * * 1-5",
      "ScheduleFrom":"2016-11-12T00:00:00.000Z",
      "ScheduleTo":"2017-01-01T00:00:00.000Z",
      "Service":"apps/com.hornbill.servicemanager/Requests",
      "API":"updateReqTimeline",
      "APIParams":{
        "0":
        {
          "Type":"Content",
          "Parameter":"requestId",
          "Content":"IN0000018"
        },
        "1":
        {
          "Type":"Content",
          "Parameter":"content",
          "Content":"This is an auto update"
        }
      }
    }
  ]
}
```

* "APIKey" - a Hornbill API key for a user account with the correct permissions to carry out all of the required API calls
* "InstanceId" - the ID of your Hornbill instance
* "Zone" - Allows you to specify the ZONE used for creating the XMLMC EndPoint URL (default is eur): https://{ZONE}api.hornbill.com/{INSTANCE}/
* "Schedule" - A JSON array, where each object within this array contains the configuration for one scheduled and repeatable task:
..* "Enabled" - set to true to enable the schedule item
..* "CronSchedule" - A Cron compatible schedule expression to schedule the API call by
..* "ScheduleFrom" - An RFC3339 formatted time string, to specify the date & time to start running any instances of the particular schedule entry. This can contain an empty string to allow you to not specify a date/time to start the schedule from
..* "ScheduleTo" - An RFC3339 formatted time string, to specify the date & time to stop running any more instances of the particular schedule entry. This can contain an empty string, should you wish the schedule to run indefinitely
..* "Service" - The Hornbill Service that contains the API you wish to running
..* "API" - The name of the API to run
..* "APIParams" - A JSON object, containing one or more other JSON objects, allowing you to specify the Order that the parameter should be presented to the API, the parameter type, the parameter ID and the content to write:
..* "Type" - Can be set to:
..* -- 'Content' - To write a parameter name (Parameter) and value (Content);
..* -- 'Open' - Allows for complex parameters to be written to the API, within the element specified in this node - must be matched with a parameter type of 'Close'
..* -- 'Close' - Allows for complex parameters to be written to the API, within the element specified in this node - must be matched with a parameter type of 'Open'
..* 'Parameter' - The name of the parameter
..* 'Content' - the string that should be written within the Parameter node

## Execute
Command Line Parameters

- file
This should point to your json configuration file and by default looks for a file in the current working directory called conf.json. If this is present you don't need to have the parameter.

'goAPIScheduler_w64.exe -file=conf.json'

When you are ready to start the scheduler:

* Open '''conf.json''' and add in the necessary configration;
* Open Command Line Prompt as Administrator;
* Change Directory to the folder containing the scheduler executables and configuration file 'C:\hornbill_scheduler\';
* Run the command relevant to your OS: goAPIScheduler_w64.exe for 64 bit Windows systems, goAPIScheduler_w32.exe for 32 bit Windows systems  
* Follow all on-screen prompts, taking careful note of all prompts and messages provided.

When the scheduler is executed, you will be presented with a list of all active schedule items from the configuration file, and these items will be executed as per the config at the relevant dates/times.

## Exit

To end the scheduler app, press CTRL+C in the Command Prompt window where the scheduler is running.

