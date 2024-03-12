package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type LogRequest struct {
	ActivityName string `json:"activity_name"`
	Duration     int    `json:"duration"`
	Memo         string `json:"memo"`
}

type SummaryRequest struct {
	ActivityName string `json:"activity_name"`
	BeginDate    string `json:"begin_date"`
	EndDate      string `json:"end_date"`
}

type ActivityItem struct {
	ActivityName string `json:"activity_name"`
	BeginTime    string `json:"begin_time"`
	Memo         string `json:"memo"`
}

const API = "http://localhost:5000"

func colorize(text, color string) string {
	colors := map[string]string{
		"reset":  "\033[0m",
		"red":    "\033[31m",
		"green":  "\033[32m",
		"yellow": "\033[33m",
		"blue":   "\033[34m",
		"purple": "\033[35m",
		"cyan":   "\033[36m",
		"white":  "\033[1;37m",
	}

	colorCode, ok := colors[color]
	if !ok {
		colorCode = colors["reset"]
	}

	return colorCode + text + colors["reset"]
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

	url := API + "/add-activity"
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

func countToNumber(count string) int {
	countNumber, err := strconv.Atoi(count)
	if err == nil {
		return countNumber
	} else {
		fmt.Println("countToNumber error:", err)
		return 0
	}
}

func formatText(title string, text string, lineLength int) string {
	words := strings.Fields(text)
	lines := []string{}
	currentLine := ""

	lines = append(lines, colorize(fmt.Sprintf("┃ %-*s ┃", lineLength, title), "yellow"))
	lines = append(lines, colorize("┣"+strings.Repeat("━", lineLength+2)+"┫", "yellow"))

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= lineLength {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	for i := 2; i < len(lines); i++ {
		lines[i] = colorize("┃", "yellow") + colorize(fmt.Sprintf(" %-*s ", lineLength, lines[i]), "white") + colorize("┃", "yellow")
	}

	border := colorize("┏"+strings.Repeat("━", lineLength+2)+"┓", "yellow")
	result := []string{border}
	result = append(result, lines...)
	result = append(result, colorize("┗"+strings.Repeat("━", lineLength+2)+"┛", "yellow"))

	return strings.Join(result, "\n")
}

func formatActivityItems(title string, items []ActivityItem, lineLength int) string {
	border := colorize("┏"+strings.Repeat("━", lineLength+2)+"┓", "yellow")
	result := []string{border, colorize(fmt.Sprintf("┃ %-*s ┃", lineLength, title), "yellow"), colorize("┣"+strings.Repeat("━", lineLength+2)+"┫", "yellow")}

	for i, item := range items {
		words := strings.Fields(item.Memo)
		lines := []string{}
		currentLine := ""

		lines = append(lines, colorize(fmt.Sprintf("┃ %-*s ┃", lineLength, item.BeginTime+", "+item.ActivityName), "yellow"))

		for _, word := range words {
			if len(currentLine)+len(word)+1 <= lineLength {
				if currentLine != "" {
					currentLine += " "
				}
				currentLine += word
			} else {
				lines = append(lines, currentLine)
				currentLine = word
			}
		}

		if currentLine != "" {
			lines = append(lines, currentLine)
		}

		for i := 1; i < len(lines); i++ {
			lines[i] = colorize("┃", "yellow") + colorize(fmt.Sprintf(" %-*s ", lineLength, lines[i]), "white") + colorize("┃", "yellow")
		}

		if i < len(items)-1 {
			lines = append(lines, colorize(fmt.Sprintf("┃ %-*s ┃", lineLength, ""), "yellow"))
		}

		result = append(result, lines...)
	}

	result = append(result, colorize("┗"+strings.Repeat("━", lineLength+2)+"┛", "yellow"))

	return strings.Join(result, "\n")
}

func buildDateRange(interval string, queryParams url.Values) (url.Values, error) {
	pattern := `([0-9]+y)?([0-9]+m)?([0-9]+w)?([0-9]+d)?`

	// Compile the pattern
	regex, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Println("Error compiling regex:", err)
		return queryParams, errors.New("bad regex")
	}

	if regex.MatchString(interval) {
		dayCount := 0
		count := ""
		for _, rune := range interval {
			if rune >= '0' && rune <= '9' {
				count += string(rune)
			} else {
				switch rune {
				case 'y':
					dayCount += 365 * countToNumber(count)
					count = ""
					break

				case 'm':
					dayCount += 30 * countToNumber(count)
					count = ""
					break

				case 'w':
					dayCount += 7 * countToNumber(count)
					count = ""
					break

				case 'd':
					dayCount += countToNumber(count)
					count = ""
					break

				}
			}
		}

		endDate := time.Now()
		beginDate := endDate.AddDate(0, 0, -dayCount)

		queryParams.Set("beginDate", beginDate.Format("2006-01-02"))
		queryParams.Set("endDate", endDate.AddDate(0, 0, 1).Format("2006-01-02"))
	} else {
		return queryParams, errors.New("bad date interval")
	}

	return queryParams, nil
}

func summaryUsage() {
	fmt.Println("Usage: summary <interval>")
	fmt.Println("where <interval> is of format #y#m#w#d, each # representing any number of digits")
}

func summary(interval string) {
	baseURL := API + "/get-summary"

	// Create URL with query parameters
	u, err := url.Parse(baseURL)
	if err != nil {
		fmt.Println(err)
		return
	}

	queryParams := u.Query()
	queryParams, err = buildDateRange(interval, queryParams)
	if err != nil {
		summaryUsage()
	}

	u.RawQuery = queryParams.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if val, ok := jsonData["response"]; ok {
		fmt.Println(formatText(fmt.Sprintf("Summary of %s to %s", queryParams.Get("beginDate"), queryParams.Get("endDate")), val.(string), 80))
	}
}

func listUsage() {
	fmt.Println("Usage: list <interval>")
	fmt.Println("where <interval> is of format #y#m#w#d, each # representing any number of digits")
}

func list(interval string) {
	baseURL := API + "/get-activities"

	// Create URL with query parameters
	u, err := url.Parse(baseURL)
	if err != nil {
		fmt.Println(err)
		return
	}

	queryParams := u.Query()
	queryParams, err = buildDateRange(interval, queryParams)
	if err != nil {
		listUsage()
	}

	u.RawQuery = queryParams.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	var jsonData map[string][]ActivityItem
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	dates := make([]string, 0, len(jsonData))
	for k := range jsonData {
		dates = append(dates, k)
	}

	sort.Strings(dates)

	for _, d := range dates {
		fmt.Println(formatActivityItems(d, jsonData[d], 80))
	}
}

func main() {
	usage := "Usage: ./ritual <command> <...args>"
	if len(os.Args) == 1 {
		fmt.Println(usage)
		return
	}

	if os.Args[1] == "log" {
		log()
	} else if os.Args[1] == "summary" {
		if len(os.Args) != 3 {
			summaryUsage()
			return
		}

		summary(os.Args[2])
	} else if os.Args[1] == "list" {
		if len(os.Args) != 3 {
			listUsage()
			return
		}

		list(os.Args[2])
	} else {
		fmt.Println("Unrecognized command " + os.Args[1] + "\n" + usage)
		return
	}
}
