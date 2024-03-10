package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
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

func countToNumber(count string) int {
	countNumber, err := strconv.Atoi(count)
	if err == nil {
		return countNumber
	} else {
		fmt.Println("countToNumber error:", err)
		return 0
	}
}

func summaryUsage() {
	fmt.Println("Usage: summary <interval>")
	fmt.Println("where <interval> is of format #y#m#w#d, each # representing any number of digits")
}

func summary(interval string) {
	pattern := `([0-9]+y)?([0-9]+m)?([0-9]+w)?([0-9]+d)?`

	// Compile the pattern
	regex, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Println("Error compiling regex:", err)
		return
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

		baseURL := "http://yourflaskappdomain.com/endpoint"
		beginDate := "2023-01-01"
		endDate := "2023-01-31"

		// Create URL with query parameters
		u, err := url.Parse(baseURL)
		if err != nil {
			fmt.Println(err)
			return
		}

		queryParams := u.Query()
		queryParams.Set("beginDate", beginDate)
		queryParams.Set("endDate", endDate)
		u.RawQuery = queryParams.Encode()

		resp, err := http.Get(u.String())
		if err != nil {
			fmt.Println(err)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Response from Flask server: %s", string(body))

		fmt.Println("got", strconv.Itoa(dayCount), "days")
	} else {
		summaryUsage()
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
	} else {
		fmt.Println("Unrecognized command " + os.Args[1] + "\n" + usage)
		return
	}
}
