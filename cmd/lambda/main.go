package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

type RequestEvent struct {
	RawPath        string `json:"rawPath"`
	RawQueryString string `json:"rawQueryString"`
}

type Response struct {
}

func HandleRequest(ctx context.Context, event *RequestEvent) (*string, error) {
	if event == nil {
		return nil, fmt.Errorf("received nil event")
	}
	message := fmt.Sprintf("RawPath: %s, RawQueryString: %s", event.RawPath, event.RawQueryString)
	log.Print(message)
	return &message, nil
}

func main() {
	lambda.Start(HandleRequest)
}
