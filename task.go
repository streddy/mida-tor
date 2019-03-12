package main

import (
	"encoding/json"
	"errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/teamnsrg/chromedp/runner"
	"io/ioutil"
)

type BrowserSettings struct {
	BrowserBinary      *string   `json:"browser_binary"`
	UserDataDirectory  *string   `json:"user_data_directory"`
	AddBrowserFlags    *[]string `json:"add_browser_flags"`
	RemoveBrowserFlags *[]string `json:"remove_browser_flags"`
	SetBrowserFlags    *[]string `json:"set_browser_flags"`
	Extensions         *[]string `json:"extensions"`
}

type CompletionSettings struct {
	CompletionCondition *string `json:"completion_condition"`
	Timeout             *int    `json:"timeout"`
	TimeAfterLoad       *int    `json:"time_after_load"`
}

type DataSettings struct {
	AllResources     *bool `json:"all_files"`
	AllScripts       *bool `json:"all_scripts"`
	JSTrace          *bool `json:"js_trace"`
	SaveRawTrace	 *bool `json:"save_raw_trace"`
	ResourceMetadata *bool `json:"resource_metadata"`
	ScriptMetadata   *bool `json:"script_metadata"`
	ResourceTree     *bool `json:"resource_tree"`
}

type OutputSettings struct {
	Path    *string `json:"path"`
	GroupID *string `json:"group_id"`
}

type MIDATask struct {
	URL *string `json:"url"`

	Browser    *BrowserSettings    `json:"browser"`
	Completion *CompletionSettings `json:"completion"`
	Data       *DataSettings       `json:"data"`
	Output     *OutputSettings     `json:"output"`

	// Track how many times we will attempt this task
	MaxAttempts *int `json:"max_attempts"`
}

type MIDATaskSet []MIDATask

type CompressedMIDATaskSet struct {
	URL *[]string `json:"url"`

	Browser    *BrowserSettings    `json:"browser"`
	Completion *CompletionSettings `json:"completion"`
	Data       *DataSettings       `json:"data"`
	Output     *OutputSettings     `json:"output"`

	// Track how many times we will attempt this task
	MaxAttempts *int `json:"max_attempts"`
}

// Single, flat struct without pointers, containing
// all info required to complete a task
type SanitizedMIDATask struct {
	Url string

	// Browser settings
	BrowserBinary     string
	UserDataDirectory string
	BrowserFlags      []runner.CommandLineOption

	// Completion Settings
	CCond         CompletionCondition
	Timeout       int
	TimeAfterLoad int

	// Data settings
	AllResources     bool
	AllScripts       bool
	JSTrace          bool
	SaveRawTrace	 bool
	ResourceMetadata bool
	ScriptMetadata   bool
	ResourceTree     bool

	// Output Settings
	OutputPath       string
	GroupID          string // For identifying experiments
	RandomIdentifier string // Randomly generated task identifier

	// Parameters for retrying a task if it fails to complete
	MaxAttempts      int
	CurrentAttempt   int
	TaskFailed       bool   // Nothing else should be done on the task once this flag is set
	FailureCode      string // Should be appended whenever a task is set to fail
	PastFailureCodes []string
}

// Reads in a single task or task list from a byte array
func ReadTasks(data []byte) ([]MIDATask, error) {
	tasks := make(MIDATaskSet, 0)
	err := json.Unmarshal(data, &tasks)
	if err == nil {
		Log.Debug("Parsed MIDATaskSet from file")
		return tasks, nil
	}

	singleTask := MIDATask{}
	err = json.Unmarshal(data, &singleTask)
	if err == nil {
		Log.Debug("Parsed single MIDATask from file")
		return append(tasks, singleTask), nil
	}

	compressedTaskSet := CompressedMIDATaskSet{}
	err = json.Unmarshal(data, &compressedTaskSet)
	if err != nil {
		return tasks, errors.New("failed to unmarshal tasks")
	}

	if compressedTaskSet.URL == nil || len(*compressedTaskSet.URL) == 0 {
		return tasks, errors.New("no URLs given in task set")
	}
	tasks = ExpandCompressedTaskSet(compressedTaskSet)

	Log.Debug("Parsed CompressedMIDATaskSet from file")
	return tasks, nil

}

// Wrapper function that reads single tasks, full task sets,
// or compressed task sets from file
func ReadTasksFromFile(fName string) ([]MIDATask, error) {
	tasks := make(MIDATaskSet, 0)

	data, err := ioutil.ReadFile(fName)
	if err != nil {
		return tasks, err
	}

	tasks, err = ReadTasks(data)
	if err != nil {
		return tasks, err
	}

	return tasks, nil
}

func ExpandCompressedTaskSet(t CompressedMIDATaskSet) []MIDATask {
	var rawTasks []MIDATask
	for _, v := range *t.URL {
		urlString := v
		newTask := MIDATask{
			URL:         &urlString,
			Browser:     t.Browser,
			Completion:  t.Completion,
			Data:        t.Data,
			Output:      t.Output,
			MaxAttempts: t.MaxAttempts,
		}
		rawTasks = append(rawTasks, newTask)
	}
	return rawTasks
}

// Retrieves raw tasks, either from a queue, file, or pre-built set
func TaskIntake(rtc chan<- MIDATask, cmd *cobra.Command, args []string) {
	if cmd.Name() == "client" {
		// TODO: Figure out how to close connection gracefully here
		taskAMQPConn, taskDeliveryChan, err := NewAMQPTasksConsumer()
		if err != nil {
			Log.Fatal(err)
		}
		defer taskAMQPConn.Shutdown()

		broadcastAMQPConn, broadcastAMQPDeliveryChan, err := NewAMQPBroadcastConsumer()
		if err != nil {
			Log.Fatal(err)
		}
		defer broadcastAMQPConn.Shutdown()

		// Remain as a client to the AMQP server until a broadcast is received which
		// causes us to exit
		breakFlag := false
		for {
			select {
			case broadcastMsg := <-broadcastAMQPDeliveryChan:
				Log.Warnf("BROADCAST RECEIVED: [ %s ]", string(broadcastMsg.Body))
				breakFlag = true
			default:
			}
			select {
			case broadcastMsg := <-broadcastAMQPDeliveryChan:
				Log.Warnf("BROADCAST RECEIVED: [ %s ]", string(broadcastMsg.Body))
				breakFlag = true
			case amqpMsg := <-taskDeliveryChan:
				rawTask, err := DecodeAMQPMessageToRawTask(amqpMsg)
				if err != nil {
					Log.Fatal(err)
				}
				rtc <- rawTask
			}
			if breakFlag {
				break
			}
		}

	} else if cmd.Name() == "file" {
		rawTasks, err := ReadTasksFromFile(viper.GetString("taskfile"))
		if err != nil {
			Log.Fatal(err)
		}

		for _, rt := range rawTasks {
			rtc <- rt
		}
	} else if cmd.Name() == "go" {
		compressedTaskSet, err := BuildCompressedTaskSet(cmd, args)
		if err != nil {
			Log.Fatal(err)
		}

		rawTasks := ExpandCompressedTaskSet(compressedTaskSet)
		for _, rt := range rawTasks {
			rtc <- rt
		}
	}

	// Start the process of closing up the pipeline and exit
	close(rtc)
}
