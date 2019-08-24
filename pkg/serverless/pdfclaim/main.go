package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/faizhasim/claim-pagerduty/pkg/pdfgenerator"
	"github.com/karrick/tparse"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse

func processAllWhatever(tparseSinceQuery, tparseUntilQuery, authtoken string) ([]string, error) {

	//authtoken := os.Getenv("PAGERDUTY_AUTH_TOKEN")
	scheduleName := os.Getenv("PAGERDUTY_SCHEDULE_NAME")
	//tparseSinceQuery := "now-3mo"
	//tparseUntilQuery := "now"
	if authtoken == "" {
		return nil, errors.New("Pagerduty auth token not set")
	}
	claimFormImageFilePath, err := pdfgenerator.DownloadPdfTemplate()
	if err != nil {
		return nil, err
	}

	since, err := tparse.ParseNow(time.RFC3339, tparseSinceQuery)
	if err != nil {
		return nil, err
	}

	until, err := tparse.ParseNow(time.RFC3339, tparseUntilQuery)
	if err != nil {
		return nil, err
	}

	entries, _ := pdfgenerator.FetchScheduleEntries(since, until, authtoken, scheduleName)

	dirPath := os.TempDir() + "/" + uuid.New().String() + "/"
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return nil, err
	}

	generatedFiles, err := pdfgenerator.MakePdfToDir(claimFormImageFilePath, dirPath, entries, time.Now())
	if err != nil {
		return nil, err
	}

	var s3PublicUrls []string
	for _, generatedFile := range generatedFiles {
		path := filepath.Base(filepath.Dir(generatedFile)) + "/" + filepath.Base(generatedFile)
		s3PublicUrls = append(s3PublicUrls, "https://"+os.Getenv("PDF_BUCKET_NAME")+".s3.amazonaws.com/"+path)
	}

	fmt.Println(":debug:", os.Getenv("AWS_REGION"), os.Getenv("PDF_BUCKET_NAME"), dirPath)

	if err := pdfgenerator.S3Upload(os.Getenv("AWS_REGION"), os.Getenv("PDF_BUCKET_NAME"), dirPath); err != nil {
		return nil, err
	}

	return s3PublicUrls, nil

}

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (Response, error) {
	since := event.QueryStringParameters["since"]
	until := event.QueryStringParameters["until"]
	pagerdutyAuthToken := event.Headers["x-pagerduty-auth-token"]

	if since == "" {
		return Response{
			StatusCode: 400, Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: "Required 'since' query string. For example: 'now-3mo'",
		}, nil
	}

	if until == "" {
		return Response{
			StatusCode: 400, Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: "Required 'until' query string. For example: '3mo'",
		}, nil
	}

	if pagerdutyAuthToken == "" {
		return Response{
			StatusCode: 400, Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: "Required 'x-pagerduty-auth-token' HTTP header. Refer to https://support.pagerduty.com/docs/generating-api-keys.'",
		}, nil
	}

	var buf bytes.Buffer

	generatedFiles, err := processAllWhatever(since, until, pagerdutyAuthToken)
	if err != nil {
		return Response{}, err
	}

	body, err := json.Marshal(generatedFiles)
	if err != nil {
		return Response{}, err
	}
	json.HTMLEscape(&buf, body)

	return Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            buf.String(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

func main() {
	lambda.Start(Handler)
}
