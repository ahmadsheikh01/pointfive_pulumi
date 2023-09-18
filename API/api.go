package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/go-github/v55/github"
)

var dynamoDBClient *dynamodb.DynamoDB
var eventCountTableName string
var actorTableName string
var reposTableName string

func main() {
	eventCountTableName = os.Getenv("EVENTS_COUNT_TABLE")
	if eventCountTableName == "" {
		fmt.Println("EVENTS_COUNT_TABLE environment variable not set")
		os.Exit(1)
	}
	fmt.Println("EVENTS_COUNT_TABLE is set to", eventCountTableName)

	actorTableName = os.Getenv("ACTORS_TABLE")
	if actorTableName == "" {
		fmt.Println("ACTORS_TABLE environment variable not set")
		os.Exit(1)
	}
	fmt.Println("ACTORS_TABLE is set to", actorTableName)

	reposTableName = os.Getenv("REPOS_TABLE")
	if reposTableName == "" {
		fmt.Println("reposTableName environment variable not set")
		os.Exit(1)
	}
	fmt.Println("reposTableName is set to", reposTableName)

	lambda.Start(handler)
}

func init() {
	// Initialize the AWS SDK and DynamoDB client
	sess := session.Must(session.NewSession())
	dynamoDBClient = dynamodb.New(sess)
}

type AppSyncResolverEvent struct {
	Field string `json:"field"`
	// Arguments map[string]interface{} `json:"arguments"`
}

type Repo struct {
	RepoURL  string `json:"repoURL"`
	RepoName string `json:"repoName"`
	RepoId   int64  `json:"repoId"`
	Stars    int    `json:"stars"`
}

type Actor struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Event struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

func getRepos() (*[]Repo, error) {
	client := github.NewClient(nil)
	input := &dynamodb.ScanInput{
		TableName: aws.String(reposTableName),
	}

	result, err := dynamoDBClient.Scan(input)
	if err != nil {
		return nil, err
	}

	if result.Items == nil {
		return nil, errors.New("repos not found")
	}

	repos := []Repo{}
	for _, i := range result.Items {
		repo := Repo{}
		err = dynamodbattribute.UnmarshalMap(i, &repo)
		if err != nil {
			continue
		}
		if repo.RepoName != "" && repo.RepoId != 0 {
			repository, _, err := client.Repositories.GetByID(context.Background(), repo.RepoId)
			if err == nil {
				repo.Stars = *repository.StargazersCount
			}
		}
		repos = append(repos, repo)
	}
	return &repos, nil
}

func getActors() (*[]Actor, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(actorTableName),
	}

	result, err := dynamoDBClient.Scan(input)
	if err != nil {
		return nil, err
	}

	if result.Items == nil {
		return nil, errors.New("actors not found")
	}

	actors := []Actor{}
	for _, i := range result.Items {
		actor := Actor{}
		err = dynamodbattribute.UnmarshalMap(i, &actor)
		if err != nil {
			continue
		}
		actors = append(actors, actor)
	}
	return &actors, nil
}

func getEvents() (*[]Event, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(eventCountTableName),
	}
	result, err := dynamoDBClient.Scan(input)
	if err != nil {
		return nil, err
	}

	if result.Items == nil {
		return nil, errors.New("events not found")
	}
	events := []Event{}
	for _, i := range result.Items {
		event := Event{}
		err = dynamodbattribute.UnmarshalMap(i, &event)
		if err != nil {
			continue
		}
		event.Type = *i["EventType"].S
		events = append(events, event)
	}

	return &events, nil
}

func handler(ctx context.Context, event json.RawMessage) (interface{}, error) {
	fmt.Println("Received event:", string(event))
	var resolverEvent AppSyncResolverEvent
	if err := json.Unmarshal(event, &resolverEvent); err != nil {
		return nil, err
	}

	fieldName := resolverEvent.Field

	switch fieldName {
	case "Repos":
		repos, err := getRepos()
		if err != nil {
			return nil, err
		}
		return repos, nil
	case "Actors":

		actors, err := getActors()
		if err != nil {
			return nil, err
		}
		return actors, nil
	case "Events":
		events, err := getEvents()
		if err != nil {
			return nil, err
		}
		return events, nil
	default:
		return nil, errors.New("invalid request")
	}
}
