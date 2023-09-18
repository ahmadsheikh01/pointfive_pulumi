package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ahmads/common"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/go-github/v55/github"
)

var githubEventsSqsUrl string

func main() {

	githubEventsSqsUrl = os.Getenv("GITHUB_CONSUMER_SQS_URL")
	if githubEventsSqsUrl == "" {
		fmt.Println("GITHUB_CONSUMER_SQS_URL is not set")
		os.Exit(1)
	}
	fmt.Println("GITHUB_CONSUMER_SQS_URL is set to", githubEventsSqsUrl)

	lambda.Start(handler)
}

func handler(ctx context.Context) error {
	events, err := fetchEvents()
	if err != nil {
		fmt.Println("failed to fetch github events", err)
		return err
	}
	err = sendEventsToConsumer(events)
	if err != nil {
		fmt.Println("failed to send events to consumer", err)
		return err
	}
	return nil
}

func fetchEvents() ([]common.Github_event, error) {
	fmt.Println("fetching github events")
	client := github.NewClient(nil)
	github_events, _, err := client.Activity.ListEvents(context.Background(), nil)

	if err != nil {
		return nil, err
	}
	var events []common.Github_event

	for _, event := range github_events {
		fmt.Println(event.Actor.GetLogin(), event.Repo.GetURL(), event.GetType(), event.Actor.GetEmail())

		event.Repo.GetName()

		tmp := common.Github_event{
			ActorLogin: event.Actor.GetLogin(),
			ActorEmail: event.Actor.GetEmail(),
			ActorName:  event.Actor.GetName(),
			RepoUrl:    event.Repo.GetURL(),
			RepoName:   event.Repo.GetName(),
			RepoId:     event.Repo.GetID(),
			EventType:  event.GetType(),
		}
		events = append(events, tmp)
	}

	fmt.Println("done fetching github events")
	return events, nil
}

func sendEventsToConsumer(events []common.Github_event) error {
	fmt.Println("sending events to consumer")
	session := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := sqs.New(session)

	for _, event := range events {

		jsonMessage, err := json.Marshal(event)

		if err != nil {
			return err
		}

		_, err = svc.SendMessage(&sqs.SendMessageInput{
			MessageBody: aws.String(string(jsonMessage)),
			QueueUrl:    &githubEventsSqsUrl,
		})

		if err != nil {
			return err
		}
	}
	fmt.Println("done sending github events to consumer")

	return nil
}
