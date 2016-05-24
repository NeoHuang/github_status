# github_status

## Usage
```
github_status [OPTIONS]
```
## OPTIONS:
    -h               Display usage
    --high           high frequency ping interval in second. is used when github is down. default is 5 seconds
    --low            low frequency ping interval in second. is used when github is normal. default is 60 seconds
    --channel        slack channel you wanna sent to.
    
get github status by pinging https://status.github.com/api/status.json. Send notification to slack channel when status changed
- slack team is required to set as Environment variable "SLACK_TEAM"
- slack token is required to set as Environment variable "SLACK_TOKEN"

## Example
	SLACK_TEAM=myteam SLACK_TOKEN=123456 github_status --high 2 --low 60 --channel github
	
