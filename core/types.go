package core

import (
	"slices"
	"strings"
	"time"
)

type ScheduledTask struct {
	Task     Task
	NextRun  time.Time
	Interval Interval
}

type Interval struct {
	Years   int `json:"years"`
	Months  int `json:"months"`
	Weeks   int `json:"weeks"`
	Days    int `json:"days"`
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
}

type Execution struct {
	Interval  Interval `json:"interval"`
	Delay     Interval `json:"delay"`
	RunBefore string   `json:"run_before"`
	RunAfter  string   `json:"run_after"`
}

type ShellTask struct {
	Name      string    `json:"name"`
	Execution Execution `json:"exection"`
	Command   string    `json:"command"`
}

type HttpTask struct {
	Name      string         `json:"name"`
	Execution Execution      `json:"exection"`
	URL       string         `json:"url"`
	Method    string         `json:"method"`
	Headers   map[string]any `json:"headers"`
	Body      map[string]any `json:"body"`
}

// if the type implements the method
// then it is of type Task
// which is what we want
type Task interface {
	GetName() string
	GetExecution() Execution
	GetCommand() string
	GetURL() string
	GetMethod() string
	GetRunBefore() Task
	GetRunAfter() Task
	GetHeaders() map[string]any
	GetBody() map[string]any
}

// shell task stuff
func (s ShellTask) GetName() string {
	return s.Name
}

func (s ShellTask) GetExecution() Execution {
	return s.Execution
}

func (s ShellTask) GetRunBefore() Task {
	if s.Execution.RunBefore == "" {
		return nil
	}

	task, _ := FetchTaskByName(s.Execution.RunBefore)
	return task
}

func (s ShellTask) GetRunAfter() Task {
	if s.Execution.RunAfter == "" {
		return nil
	}

	task, _ := FetchTaskByName(s.Execution.RunAfter)
	return task
}

func (s ShellTask) GetCommand() string {
	return s.Command
}

func (s ShellTask) GetURL() string {
	return "not-available"
}

func (s ShellTask) GetMethod() string {
	return "not-available"
}

func (s ShellTask) GetBody() map[string]any {
	return map[string]any{}
}

func (s ShellTask) GetHeaders() map[string]any {
	return map[string]any{}
}

// http task stuff
func (h HttpTask) GetName() string {
	return h.Name
}

func (h HttpTask) GetExecution() Execution {
	return h.Execution
}

func (h HttpTask) GetRunBefore() Task {
	if h.Execution.RunBefore == "" {
		return nil
	}

	task, _ := FetchTaskByName(h.Execution.RunBefore)
	return task
}

func (h HttpTask) GetRunAfter() Task {
	if h.Execution.RunAfter == "" {
		return nil
	}

	task, _ := FetchTaskByName(h.Execution.RunAfter)
	return task
}

func (h HttpTask) GetCommand() string {
	return "not-available"
}

func (h HttpTask) GetURL() string {
	return h.URL
}

func (h HttpTask) GetMethod() string {
	if enforceMethods(h.Method) {
		return h.Method
	}

	return "not-available"
}

func (h HttpTask) GetBody() map[string]any {
	return h.Body
}

func (h HttpTask) GetHeaders() map[string]any {
	return h.Headers
}

// just making sure that the methods align with the
// http standars
// also uppercasing the method input
// to be parameter safe should the user use
// "get" instead of "GET" in the json config
func enforceMethods(method string) bool {
	availableMethods := []string{METHOD_GET, METHOD_POST, METHOD_PUT, METHOD_PATCH, METHOD_DELETE, METHOD_HEAD}
	return slices.Contains(availableMethods, strings.ToUpper(method))
}
