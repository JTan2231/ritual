package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
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

type ChatRequest struct {
	Chat string `json:"chat"`
}

type SummaryRequest struct {
	ActivityName string `json:"activity_name"`
	BeginDate    string `json:"begin_date"`
	EndDate      string `json:"end_date"`
}

type SignupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ActivityListItem struct {
	ActivityName string `json:"activity_name"`
	BeginTime    string `json:"activity_begin"`
	EndTime      string `json:"activity_end"`
	Duration     string
	Memo         string `json:"memo"`
}

const API = "https://ritual-api-production.up.railway.app"

// TODO: there's a lot of repeated code here; clean up requests into shared functions

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
		fmt.Println("Usage: ./ritual log <activity_name> <duration> <message>")
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

	username, hasUser := os.LookupEnv("RITUAL_USERNAME")
	password, hasPass := os.LookupEnv("RITUAL_PASSWORD")

	if !hasUser || !hasPass {
		fmt.Println("Both the $RITUAL_USERNAME and $RITUAL_PASSWORD environment variables need set for this command")
		return
	}

	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
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

	if res.StatusCode != 200 {
		fmt.Printf("Request failed with status code: %d\n", res.StatusCode)
		fmt.Println("Response message:", string(body))
		return
	}

	fmt.Println(string(body))
}

// TODO: This should handle date trimming, not the backend
func displayActivityListItems(jsonData map[string][]ActivityListItem) {
	dates := make([]string, 0, len(jsonData))
	for k := range jsonData {
		dates = append(dates, k)
	}

	sort.Strings(dates)

	timeFormat := "15:04:05"
	for _, d := range dates {
		day := jsonData[d]
		for i, activity := range day {
			t1, _ := time.Parse(timeFormat, activity.BeginTime)
			t2, _ := time.Parse(timeFormat, activity.EndTime)

			activity.Duration = fmt.Sprintf("%d minutes", int(t2.Sub(t1).Minutes()))
			activity.BeginTime = activity.BeginTime[:len(timeFormat)-3]
			activity.EndTime = activity.EndTime[:len(timeFormat)-3]

			day[i] = activity
		}

		fmt.Println(formatActivityListItems(d, day, 80))
	}
}

// TODO: what if we're not given a time or duration?
func chat() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: ./ritual chat \"your chat message\"")
		return
	}

	payload := ChatRequest{
		Chat: os.Args[2],
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		return
	}

	url := API + "/chat"
	method := "POST"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Println(err)
		return
	}

	username, hasUser := os.LookupEnv("RITUAL_USERNAME")
	password, hasPass := os.LookupEnv("RITUAL_PASSWORD")

	if !hasUser || !hasPass {
		fmt.Println("Both the $RITUAL_USERNAME and $RITUAL_PASSWORD environment variables need set for this command")
		return
	}

	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
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

	if res.StatusCode != 200 {
		fmt.Printf("Request failed with status code: %d\n", res.StatusCode)
		fmt.Println("Response message:", string(body))
		return
	}

	var jsonData map[string][]ActivityListItem
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	displayActivityListItems(jsonData)

	activityItems := make([]ActivityListItem, 0)
	for _, value := range jsonData {
		// we took away the seconds earlier, now we gotta add them back so the backend doesn't throw a fit
		for i, v := range value {
			v.BeginTime += ":00"
			v.EndTime += ":00"

			value[i] = v
		}

		activityItems = append(activityItems, value...)
	}

	var response string
	fmt.Print(colorize("Does this look correct? y/n: ", "white"))
	fmt.Scanln(&response)

	if response == "y" || response == "Y" {
		jsonPayload, err := json.Marshal(activityItems)
		if err != nil {
			fmt.Println(err)
			return
		}

		url = API + "/add-activities"

		// TODO: lotta repeated code (again)
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonPayload))
		if err != nil {
			fmt.Println(err)
			return
		}

		username, hasUser := os.LookupEnv("RITUAL_USERNAME")
		password, hasPass := os.LookupEnv("RITUAL_PASSWORD")

		if !hasUser || !hasPass {
			fmt.Println("Both the $RITUAL_USERNAME and $RITUAL_PASSWORD environment variables need set for this command")
			return
		}

		req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
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

		if res.StatusCode != 200 {
			fmt.Printf("Request failed with status code: %d\n", res.StatusCode)
			fmt.Println("Response message:", string(body))
			return
		}

		fmt.Println(string(body))
	} else {
		fmt.Println("Incorrect format")
	}
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

func formatActivityListItems(title string, items []ActivityListItem, lineLength int) string {
	border := colorize("┏"+strings.Repeat("━", lineLength+2)+"┓", "yellow")
	result := []string{border, colorize(fmt.Sprintf("┃ %-*s ┃", lineLength, title), "yellow"), colorize("┣"+strings.Repeat("━", lineLength+2)+"┫", "yellow")}

	for i, item := range items {
		words := strings.Fields(item.Memo)
		lines := []string{}
		currentLine := ""

		lines = append(lines, colorize(fmt.Sprintf("┃ %-*s ┃", lineLength, item.BeginTime+", "+item.ActivityName+" - "+item.Duration), "yellow"))

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

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	username, hasUser := os.LookupEnv("RITUAL_USERNAME")
	password, hasPass := os.LookupEnv("RITUAL_PASSWORD")

	if !hasUser || !hasPass {
		fmt.Println("Both the $RITUAL_USERNAME and $RITUAL_PASSWORD environment variables need set for this command")
		return
	}

	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))

	client := &http.Client{}
	resp, err := client.Do(req)
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

	if resp.StatusCode != 200 {
		fmt.Printf("Request failed with status code: %d\n", resp.StatusCode)
		fmt.Println("Response message:", string(body))
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

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	username, hasUser := os.LookupEnv("RITUAL_USERNAME")
	password, hasPass := os.LookupEnv("RITUAL_PASSWORD")

	if !hasUser || !hasPass {
		fmt.Println("Both the $RITUAL_USERNAME and $RITUAL_PASSWORD environment variables need set for this command")
		return
	}

	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))

	client := &http.Client{}
	resp, err := client.Do(req)
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

	if resp.StatusCode != 200 {
		fmt.Printf("Request failed with status code: %d\n", resp.StatusCode)
		fmt.Println("Response message:", string(body))
		return
	}

	var jsonData map[string][]ActivityListItem
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	displayActivityListItems(jsonData)
}

func signupUsage() {
	fmt.Println(colorize("Usage: ", "yellow") + "./ritual signup <email> <password>")
}

func signup(username string, password string) {
	if len(os.Args) != 4 {
		fmt.Println("Usage: ./ritual signup <email> <password>")
		return
	}

	_, err := mail.ParseAddress(username)
	if err != nil {
		fmt.Println(colorize("Error: ", "yellow") + "Invalid email address")
		return
	}

	payload := SignupRequest{
		Username: username,
		Password: password,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		return
	}

	url := API + "/create-account"
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

func help() {
	fmt.Println(colorize("Usage:", "yellow"))
	fmt.Println("  ./ritual <command> [options]")
	fmt.Println()
	fmt.Println(colorize("Commands:", "yellow"))
	fmt.Println("  log       Log an activity")
	fmt.Println("  summary   Get a summary of activities for a given interval")
	fmt.Println("  list      List activities for a given interval")
	fmt.Println()
	fmt.Println(colorize("Options:", "yellow"))
	fmt.Println("  log:")
	fmt.Println("    <activity_name>   Name of the activity")
	fmt.Println("    <duration>        Duration of the activity (in minutes)")
	fmt.Println("    <message>         Memo or description of the activity")
	fmt.Println()
	fmt.Println("  summary:")
	fmt.Println("    <interval>        Interval for the summary (e.g., 1y2m3w4d)")
	fmt.Println()
	fmt.Println("  list:")
	fmt.Println("    <interval>        Interval for listing activities (e.g., 1y2m3w4d)")
	fmt.Println()
	fmt.Println(colorize("Interval Format:", "yellow"))
	fmt.Println("  The <interval> argument should be in the format #y#m#w#d, where:")
	fmt.Println("    #y represents the number of years")
	fmt.Println("    #m represents the number of months")
	fmt.Println("    #w represents the number of weeks")
	fmt.Println("    #d represents the number of days")
	fmt.Println("  Each # can be any number of digits.")
	fmt.Println()
	fmt.Println(colorize("Examples:", "yellow"))
	fmt.Println("  ./ritual log coding 120 \"Worked on the CLI tool\"")
	fmt.Println("  ./ritual summary 1m2w")
	fmt.Println("  ./ritual list 2w3d")
}

func main() {
	if len(os.Args) < 2 {
		help()
		return
	}

	switch os.Args[1] {
	case "log":
		log()

	case "summary":
		if len(os.Args) != 3 {
			summaryUsage()
			return
		}

		summary(os.Args[2])

	case "list":
		if len(os.Args) != 3 {
			listUsage()
			return
		}

		list(os.Args[2])

	case "help":
		help()

	case "signup":
		if len(os.Args) != 4 {
			signupUsage()
			return
		}

		signup(os.Args[2], os.Args[3])

	case "chat":
		chat()

	default:
		fmt.Println("Unrecognized command " + colorize(os.Args[1], "yellow") + "\n")
		help()
		return
	}
}
