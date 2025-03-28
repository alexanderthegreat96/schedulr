package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

type Task struct {
	Type    string `json:"type"`
	Command string `json:"command,omitempty"`
	URL     string `json:"url,omitempty"`
	Method  string `json:"method,omitempty"`
}

var runCmd = &cobra.Command{
	Use:   "run [task_type] [task_id]",
	Short: "Runs a specified task",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		taskType, taskID := args[0], args[1]
		taskPath := fmt.Sprintf("tasks/%s/%s.json", taskType, taskID)

		data, err := ioutil.ReadFile(taskPath)
		if err != nil {
			fmt.Println("Error reading task file:", err)
			return
		}

		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			fmt.Println("Error parsing task JSON:", err)
			return
		}

		switch taskType {
		case "shell":
			runShellTask(task)
		case "http":
			runHTTPTask(task)
		default:
			fmt.Println("Unsupported task type:", taskType)
		}
	},
}

func runShellTask(task Task) {
	fmt.Println("Executing shell command:", task.Command)
	cmd := exec.Command("sh", "-c", task.Command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error executing shell command:", err)
	}
}

func runHTTPTask(task Task) {
	fmt.Println("Making HTTP request:", task.Method, task.URL)
	req, err := http.NewRequest(task.Method, task.URL, nil)
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error executing HTTP request:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)
}

func init() {
	rootCmd.AddCommand(runCmd)
}
