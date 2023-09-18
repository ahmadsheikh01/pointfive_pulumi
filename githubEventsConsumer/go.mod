module githubEventsConsumer

go 1.21.1

require github.com/aws/aws-lambda-go v1.41.0

require github.com/ahmads/common v0.0.0

require (
	github.com/aws/aws-sdk-go v1.45.11 // indirect
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.10.39 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.21.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.15.5 // indirect
	github.com/aws/smithy-go v1.14.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
)

replace github.com/ahmads/common => ../common
