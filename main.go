package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	good    = "good"
	minor   = "minor"
	major   = "major"
	unknown = "unknown"

	githubStatusApi = "https://status.github.com/api/status.json"
	version         = "1.0.0"
	usageMessage    = `Usage: github_status [OPTIONS]

OPTIONS:
    -h               Display usage
    --high           high frequency ping interval (example "1s" "5m" "1.5h"). is used when github is down. default is 5 seconds
    --low            low frequency ping interval (example "1s" "5m" "1.5h"). is used when github is normal. default is 1 minute
    --channel        slack channel you wanna sent to.
    --verbose        output verbose log

get github status by pinging https://status.github.com/api/status.json. Send notification to slack channel when status changed
slack team is required to set as Environment variable "SLACK_TEAM"
slack token is required to set as Environment variable "SLACK_TOKEN"

Example:
	SLACK_TEAM=myteam SLACK_TOKEN=123456 github_status --high 2s --low 1m --channel github
`
)

type githubStatus struct {
	Status      string    `json:"status"`
	LastUpdated time.Time `json:"last_updated"`
}

var (
	lowFrequency  time.Duration
	highFrequency time.Duration
	channel       string
	verbose       bool
)

func main() {
	flag.Usage = func() {
		printVersion()
		fmt.Fprintln(os.Stderr, usageMessage)
		return
	}

	var low, high string
	flag.StringVar(&low, "low", "1m", "low frequency")
	flag.StringVar(&high, "high", "5s", "high frequency")
	flag.StringVar(&channel, "channel", "", "channel name in slack")
	flag.BoolVar(&verbose, "verbose", false, "output verbose log")
	flag.Parse()

	if verbose {
		log.Printf("high frequency: %v, low frequency: %v", low, high)
	}
	var err error
	lowFrequency, err = time.ParseDuration(low)
	if err != nil {
		log.Fatal("failed to parse low")
	}
	highFrequency, err = time.ParseDuration(high)
	if err != nil {
		log.Fatal("failed to parse high")
	}

	var lastStatus = loadLastStatus()
	ticker := newTicker(lastStatus, lowFrequency, highFrequency)
	status, ticker := checkGithubStatus(lastStatus, ticker)
	for {
		<-ticker.C
		status, ticker = checkGithubStatus(status, ticker)
	}
}

func checkGithubStatus(lastStatus string, ticker *time.Ticker) (status string, nextTicker *time.Ticker) {
	status = getStatus()
	if verbose {
		log.Printf("get status: %s", status)
	}
	if status != lastStatus {
		ticker.Stop()
		ticker = newTicker(status, lowFrequency, highFrequency)

		log.Printf("status changed to %q", status)
		saveLastStatus(status)
		sendSlackNotification(channel, status)
	}
	return status, ticker
}

func getStatus() string {
	if verbose {
		log.Printf("ping github status api")
	}

	resp, err := http.Get(githubStatusApi)
	if err != nil {
		log.Printf("error ping github api")
		return unknown
	}
	defer resp.Body.Close()
	var status githubStatus
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &status)
	if err != nil {
		log.Printf("error unmarshal github status response")
		return unknown
	}

	return status.Status
}

func newTicker(status string, low, high time.Duration) *time.Ticker {
	if status == good {
		if verbose {
			log.Printf("new ticker with low frequency")
		}
		return time.NewTicker(low)
	}

	log.Printf("new ticker with high frequency")
	return time.NewTicker(high)
}

func loadLastStatus() string {
	bytes, err := ioutil.ReadFile("last")
	if err != nil {
		log.Printf("error reading last status")
		return ""
	}
	lastStatus := string(bytes)
	fmt.Printf("loaded last status: %q\n", lastStatus)
	return lastStatus
}

func saveLastStatus(lastStatus string) {
	err := ioutil.WriteFile("last", []byte(lastStatus), 0644)
	if err != nil {
		log.Printf("error saving last status")
	}
}

func printVersion() {
	fmt.Printf("github status monitor Ver:%s\n", version)
	fmt.Println("Author:Neo")
}

func sendSlackNotification(channel string, lastStatus string) {
	if channel == "" {
		return
	}

	slackTeam := os.Getenv("SLACK_TEAM")
	if slackTeam == "" {
		return
	}

	slackToken := os.Getenv("SLACK_TOKEN")
	if slackToken == "" {
		return
	}

	channel = "%23" + channel // #channel
	requestUrl := fmt.Sprintf("https://%s.slack.com/services/hooks/slackbot?token=%s&channel=%s", slackTeam, slackToken, channel)
	switch lastStatus {
	case good:
		postSlack(requestUrl, "github is now all good :white_check_mark:")
	case minor:
		postSlack(requestUrl, "github has minor issue :construction:")
	case major:
		postSlack(requestUrl, "github is DOWN!!!! :x:")
	}
}

func postSlack(requestUrl, msg string) {
	resp, err := http.Post(requestUrl, "text/plain", strings.NewReader(msg))
	defer resp.Body.Close()
	if err != nil {
		log.Printf("error send to slack %s", err)
	}
}
