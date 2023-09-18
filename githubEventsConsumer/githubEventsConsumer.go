package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ahmads/common"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var eventCountTableName string
var actorTableName string
var reposTableName string

var db *dynamodb.DynamoDB

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

	initDynamoDb()
	lambda.Start(handler)
}

func initDynamoDb() error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}
	db = dynamodb.New(sess)
	return nil
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	for _, record := range sqsEvent.Records {
		// Process each SQS message
		fmt.Printf("Message ID: %s\n", record.MessageId)
		fmt.Printf("Message Body: %s\n", record.Body)
		var githubEvent common.Github_event
		err := json.Unmarshal([]byte(record.Body), &githubEvent)
		fmt.Println("Message :", githubEvent)

		if err != nil {
			fmt.Println("Error:", err)
			return err
		}
		handleEvent(githubEvent)
	}
	return nil
}

func handleEvent(event common.Github_event) {
	createOrUpdateEventCount(event.EventType)
	createOrUpdateActor(event)
	createOrUpdateRepo(event)
}

func createOrUpdateEventCount(eventType string) error {

	incrementValue := 1
	conditionExpression := "attribute_exists(EventType)"
	updateExpression := "SET #count = #count + :increment"
	expressionAttributeNames := map[string]*string{
		"#count": aws.String("Count"),
	}
	expressionAttributeValues := map[string]*dynamodb.AttributeValue{
		":increment": {
			N: aws.String(fmt.Sprintf("%d", incrementValue)),
		},
	}

	// Attempt to update the existing record
	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(eventCountTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"EventType": {S: aws.String(eventType)},
		},
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeNames:  expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
		ConditionExpression:       aws.String(conditionExpression),
		ReturnValues:              aws.String("NONE"),
	}

	_, err := db.UpdateItem(updateInput)
	if err == nil {
		fmt.Println("Update successful")
		return nil
	} else {
		// If the condition fails, create the record with an initial 'count' value of 1
		putInput := &dynamodb.PutItemInput{
			TableName: aws.String(eventCountTableName),
			Item: map[string]*dynamodb.AttributeValue{
				"EventType": {S: aws.String(eventType)},
				"Count":     {N: aws.String(fmt.Sprintf("%d", incrementValue))},
			},
		}

		_, err := db.PutItem(putInput)
		if err != nil {
			fmt.Println("Error:", err)
			return err
		}
		fmt.Println("Record created")
	}
	return nil
}

func createOrUpdateActor(event common.Github_event) error {

	updateExpression := "SET LastAction = :lastAction, Email = :email, ActorName = :name"

	expressionAttributeValues := map[string]*dynamodb.AttributeValue{
		":lastAction": {
			N: aws.String(fmt.Sprintf("%d", time.Now().Unix())),
		},
		":email": {
			S: aws.String(event.ActorEmail),
		},
		":name": {
			S: aws.String(event.ActorName),
		},
	}

	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(actorTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Login": {S: aws.String(event.ActorLogin)},
		},
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionAttributeValues,
		ReturnValues:              aws.String("NONE"),
	}

	_, err := db.UpdateItem(updateInput)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}
	return nil
}

func createOrUpdateRepo(event common.Github_event) error {

	updateExpression := "SET RepoName = :repoName, RepoId = :repoId"

	expressionAttributeValues := map[string]*dynamodb.AttributeValue{
		":repoName": {
			S: aws.String(event.RepoName),
		},
		":repoId": {
			N: aws.String(fmt.Sprintf("%d", event.RepoId)),
		},
	}

	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(reposTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"RepoUrl": {S: aws.String(event.RepoUrl)},
		},
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionAttributeValues,
		ReturnValues:              aws.String("NONE"),
	}

	_, err := db.UpdateItem(updateInput)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}
	return nil
}
