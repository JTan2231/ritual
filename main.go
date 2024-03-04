package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

type LogRequest struct {
	ActivityName string `json:"activity_name"`
	Duration     int    `json:"duration"`
	Memo         string `json:"memo"`
}

func log() {
	if len(os.Args) != 5 {
		fmt.Println("Usage: ./cli log <activity_name> <duration> <message>")
		return
	}

	activityName := os.Args[2]
	duration := 0
	if value, err := strconv.Atoi(os.Args[3]); err == nil {
		duration = value
	} else {
		fmt.Println("Error: duration should be a number, got " + os.Args[3])
		return
	}

	memo := os.Args[4]

	payload := LogRequest{
		ActivityName: activityName,
		Duration:     duration,
		Memo:         memo,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		return
	}

	url := "http://localhost:5000/add-activity"
	method := "POST"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(body))
}

func main() {
	usage := "Usage: ./ritual <command> <...args>"
	if len(os.Args) == 1 {
		fmt.Println(usage)
		return
	}

	if os.Args[1] == "log" {
		log()
	} else {
		fmt.Println("Unrecognized command " + os.Args[1] + "\n" + usage)
		return
	}
}
