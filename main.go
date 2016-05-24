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
	good  = "good"
	minor = "minor"
	major = "major"

	githubStatusApi = "https://status.github.com/api/status.json"
	version         = "1.0.0"
	usageMessage    = `Usage: github_status [OPTIONS]

OPTIONS:
    -h               Display usage
    --high           high frequency ping interval in second. is used when github is down. default is 5 seconds
    --low            low frequency ping interval in second. is used when github is normal. default is 60 seconds
    --channel        slack channel you wanna sent to.

get github status by pinging https://status.github.com/api/status.json. Send notification to slack channel when status changed
slack team is required to set as Environment variable "SLACK_TEAM"
slack token is required to set as Environment variable "SLACK_TOKEN"

Example:
	SLACK_TEAM=myteam SLACK_TOKEN=123456 github_status --high 2 --low 60 --channel github
`
)

type githubStatus struct {
	Status      string    `json:"status"`
	LastUpdated time.Time `json:"last_updated"`
}

func main() {
	flag.Usage = func() {
		printVersion()
		fmt.Fprintln(os.Stderr, usageMessage)
		return
	}

	var low int
	var high int
	var channel string
	flag.IntVar(&low, "low", 60, "low frequency")
	flag.IntVar(&high, "high", 5, "high frequency")
	flag.StringVar(&channel, "channel", "", "channel name in slack")
	flag.Parse()

	lowFrequency := time.Second * time.Duration(low)
	highFrequency := time.Second * time.Duration(high)

	var lastStatus = loadLastStatus()
	timer := time.NewTimer(lowFrequency)
	for {
		status := getStatus()
		if status.Status != good {
			timer.Reset(highFrequency)
		} else {
			timer.Reset(lowFrequency)
		}
		if status.Status != lastStatus {
			fmt.Printf("status is %s\n", status.Status)
			lastStatus = status.Status
			saveLastStatus(lastStatus)
			if channel != "" {
				sendSlackNotification(channel, lastStatus)
			}
		}
		<-timer.C
	}
}

func getStatus() *githubStatus {
	resp, err := http.Get(githubStatusApi)
	if err != nil {
		log.Printf("error ping github api")
		return nil
	}
	defer resp.Body.Close()
	var status githubStatus
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &status)
	if err != nil {
		log.Printf("error unmarshal github status response")
		return nil
	}

	return &status
}

func loadLastStatus() string {
	bytes, err := ioutil.ReadFile("last")
	if err != nil {
		log.Printf("error reading last status")
		return ""
	}
	return string(bytes)
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
	slackTeam := os.Getenv("SLACK_TEAM")
	slackToken := os.Getenv("SLACK_TOKEN")
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
