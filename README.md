# Generate claim form prefilled from PagerDuty API


## As cli

    go get github.com/faizhasim/gdp-pagerduty-claim/cmd/claimpd

## On Serverless

### Poor Man Deployment + Test

    make deploy && curl -v -H 'x-pagerduty-auth-token: xxxxx' https://3ognk3u3q0.execute-api.us-east-1.amazonaws.com/dev/pdfclaim/orion\?since\=now-3mo\&until\=now && sls logs -f pdfclaim -t

