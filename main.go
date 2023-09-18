package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/appsync"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/sqs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create an IAM Role for our Lambda to assume
		lambdaRole, err := iam.NewRole(ctx, "lambdaRole", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Action": "sts:AssumeRole",
						"Principal": {
							"Service": "lambda.amazonaws.com"
						},
						"Effect": "Allow",
						"Sid": ""
					},
					{
					"Action": "sts:AssumeRole",
					"Principal": {
						"Service": "appsync.amazonaws.com"
					},
					"Effect": "Allow",
					"Sid": ""
				}
				]
			}`),
		})
		if err != nil {
			return err
		}

		lambdaLogPolicy, err := iam.NewPolicy(ctx, "lambdaLogPolicy", &iam.PolicyArgs{
			Policy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": [
						"logs:CreateLogGroup",
						"logs:CreateLogStream",
						"logs:PutLogEvents"
					],
					"Resource": "arn:aws:logs:*:*:*"
				}]
			}`),
		})

		if err != nil {
			return err
		}

		additionalResourcesPolicy, err := iam.NewPolicy(ctx, "additionalResourcesPolicy", &iam.PolicyArgs{
			Policy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": [
						"sqs:*"
					],
					"Resource": "*"
				},
				{
					"Effect": "Allow",
					"Action": [
						"dynamodb:*"
					],
					"Resource": "*"
				},
				{
					"Effect": "Allow",
					"Action": [
						"lambda:*"
					],
					"Resource": "*"
				}
				]
			}`),
		})

		if err != nil {
			return err
		}

		// Create DynamoDB table
		actorsTable, err := dynamodb.NewTable(ctx, "actorsTable", &dynamodb.TableArgs{
			Attributes: dynamodb.TableAttributeArray{
				&dynamodb.TableAttributeArgs{
					Name: pulumi.String("Login"),
					Type: pulumi.String("S"),
				},
				dynamodb.TableAttributeArgs{
					Name: pulumi.String("LastAction"),
					Type: pulumi.String("N"),
				},
			},
			HashKey: pulumi.String("Login"),
			GlobalSecondaryIndexes: dynamodb.TableGlobalSecondaryIndexArray{
				&dynamodb.TableGlobalSecondaryIndexArgs{
					Name:           pulumi.String("LastActionIndex"),
					HashKey:        pulumi.String("LastAction"),
					ProjectionType: pulumi.String("ALL"),
				},
			},
			BillingMode: pulumi.String("PAY_PER_REQUEST"),
			TableClass:  pulumi.String("STANDARD"),
		})

		if err != nil {
			return err
		}

		eventCountTable, err := dynamodb.NewTable(ctx, "EventsCounts", &dynamodb.TableArgs{
			Attributes: dynamodb.TableAttributeArray{
				&dynamodb.TableAttributeArgs{
					Name: pulumi.String("EventType"),
					Type: pulumi.String("S"),
				},
			},
			HashKey:     pulumi.String("EventType"),
			BillingMode: pulumi.String("PAY_PER_REQUEST"),
			TableClass:  pulumi.String("STANDARD"),
		})

		if err != nil {
			return err
		}

		reposTable, err := dynamodb.NewTable(ctx, "ReposTable", &dynamodb.TableArgs{
			Attributes: dynamodb.TableAttributeArray{
				&dynamodb.TableAttributeArgs{
					Name: pulumi.String("RepoUrl"),
					Type: pulumi.String("S"),
				},
			},
			HashKey:     pulumi.String("RepoUrl"),
			BillingMode: pulumi.String("PAY_PER_REQUEST"),
			TableClass:  pulumi.String("STANDARD"),
		})

		if err != nil {
			return err
		}

		// Create SQS github_event_consumer_sqs
		github_event_consumer_sqs, err := sqs.NewQueue(ctx, "githubConsumerSQS", &sqs.QueueArgs{})
		if err != nil {
			return err
		}
		// Create fetcher Lambda
		githubEventsFetcher, err := lambda.NewFunction(ctx, "githubEventsFetcher", &lambda.FunctionArgs{
			Runtime: lambda.RuntimeGo1dx,
			Code:    pulumi.NewFileArchive("./tmp/githubEventsFetcher.zip"),
			Handler: pulumi.String("githubEventsFetcher"),
			Role:    lambdaRole.Arn,
			Environment: &lambda.FunctionEnvironmentArgs{
				Variables: pulumi.StringMap{
					"GITHUB_CONSUMER_SQS_URL": github_event_consumer_sqs.Url,
				},
			},
		})

		if err != nil {
			return err
		}

		_, err = iam.NewRolePolicyAttachment(ctx, "githubEventsFetcherLogAttachment", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaRole.Name,
			PolicyArn: lambdaLogPolicy.Arn,
		})

		if err != nil {
			return err
		}

		_, err = iam.NewRolePolicyAttachment(ctx, "additionalPolicyAttachment", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaRole.Name,
			PolicyArn: additionalResourcesPolicy.Arn,
		})

		if err != nil {
			return err
		}

		// Set up an AWS CloudWatch event rule to trigger the Lambda function every 10 minutes
		rule, err := cloudwatch.NewEventRule(ctx, "everyTenMinutes", &cloudwatch.EventRuleArgs{
			ScheduleExpression: pulumi.String("rate(1000 minutes)"),
		})
		if err != nil {
			return err
		}

		// Attach the rule to the Lambda function
		_, err = cloudwatch.NewEventTarget(ctx, "everyTenMinutes", &cloudwatch.EventTargetArgs{
			Rule:     rule.Name,
			TargetId: pulumi.String("githubEventsFetcher"),
			Arn:      githubEventsFetcher.Arn,
		}, pulumi.DependsOn([]pulumi.Resource{githubEventsFetcher, rule}))

		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "allowTriggerGithubEventsFetcherLambda", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  githubEventsFetcher.Name,
			Principal: pulumi.String("events.amazonaws.com"),
			SourceArn: rule.Arn,
		}, pulumi.DependsOn([]pulumi.Resource{githubEventsFetcher, rule}))

		if err != nil {
			return err
		}

		githubEventsConsumer, err := lambda.NewFunction(ctx, "githubEventsConsumer", &lambda.FunctionArgs{
			Runtime: lambda.RuntimeGo1dx,
			Code:    pulumi.NewFileArchive("./tmp/githubEventsConsumer.zip"),
			Handler: pulumi.String("githubEventsConsumer"),
			Role:    lambdaRole.Arn,
			Environment: &lambda.FunctionEnvironmentArgs{
				Variables: pulumi.StringMap{
					"ACTORS_TABLE":       actorsTable.Name,
					"EVENTS_COUNT_TABLE": eventCountTable.Name,
					"REPOS_TABLE":        reposTable.Name,
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{github_event_consumer_sqs, actorsTable, reposTable, eventCountTable}))
		if err != nil {
			return err
		}

		// Enable triggering of the second Lambda function by the SQS queue
		if _, err := lambda.NewEventSourceMapping(ctx, "invokeGithubEventsConsumerLambda", &lambda.EventSourceMappingArgs{
			EventSourceArn: github_event_consumer_sqs.Arn,
			FunctionName:   githubEventsConsumer.Name,
		}, pulumi.DependsOn([]pulumi.Resource{githubEventsConsumer})); err != nil {
			return err
		}

		_, err = iam.NewRolePolicyAttachment(ctx, "githubEventsConsumerLogAttachment", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaRole.Name,
			PolicyArn: lambdaLogPolicy.Arn,
		})

		if err != nil {
			return err
		}

		resolverLambdaFunction, err := lambda.NewFunction(ctx, "resolverLambdaFunction", &lambda.FunctionArgs{
			Runtime: lambda.RuntimeGo1dx,
			Code:    pulumi.NewFileArchive("./tmp/api.zip"),
			Handler: pulumi.String("api"),
			Role:    lambdaRole.Arn,
			Environment: &lambda.FunctionEnvironmentArgs{
				Variables: pulumi.StringMap{
					"ACTORS_TABLE":       actorsTable.Name,
					"EVENTS_COUNT_TABLE": eventCountTable.Name,
					"REPOS_TABLE":        reposTable.Name,
				},
			},
			Timeout: pulumi.Int(500),
		}, pulumi.DependsOn([]pulumi.Resource{github_event_consumer_sqs, actorsTable, reposTable, eventCountTable}))

		if err != nil {
			return err
		}

		api, err := appsync.NewGraphQLApi(ctx, "api", &appsync.GraphQLApiArgs{
			Schema: pulumi.String(`
                type Repo {
                  repoURL: String
                  repoName: String
									repoId: Int
                  stars: Int
                }
                
                type Actor {
                  login: String
                  name: String
                  email: String
                }

                type Event {
                  type: String
                  count: Int
                }

                type Query {
                  Repos: [Repo]
                  Actors: [Actor]
                  Events: [Event]
                }`),
			AuthenticationType: pulumi.String("API_KEY"),
		})

		if err != nil {
			return err
		}

		// Generate an API Key for the newly created AppSync API for public access.
		_, err = appsync.NewApiKey(ctx, "myApiKey", &appsync.ApiKeyArgs{
			ApiId:   api.ID(),                              // Associate to the new API
			Expires: pulumi.String("2024-08-31T00:00:00Z"), // Set the expiry of this key. Adjust as necessary.
		})

		if err != nil {
			return err
		}

		dataSource, err := appsync.NewDataSource(ctx, "dataSource", &appsync.DataSourceArgs{
			ApiId: api.ID(),
			Name:  pulumi.String("lambda"),
			Type:  pulumi.String("AWS_LAMBDA"),
			LambdaConfig: &appsync.DataSourceLambdaConfigArgs{
				FunctionArn: resolverLambdaFunction.Arn,
			},
			ServiceRoleArn: lambdaRole.Arn,
		}, pulumi.DependsOn([]pulumi.Resource{resolverLambdaFunction, api}))
		if err != nil {
			return err
		}

		fields := []string{"Repos", "Actors", "Events"}
		for _, field := range fields {
			_, err = appsync.NewResolver(ctx, "resolver_"+field, &appsync.ResolverArgs{
				ApiId:      api.ID(),
				Type:       pulumi.String("Query"),
				Field:      pulumi.String(field),
				DataSource: dataSource.Name,
				RequestTemplate: pulumi.String(`{
                    "version": "2017-02-28",
                    "operation": "Invoke",
                    "payload": {
                      "field": "` + field + `",
                    }
                  }`),
				ResponseTemplate: pulumi.String("$util.toJson($context.result)"),
			})
			if err != nil {
				return err
			}
		}

		ctx.Export("githubEventsFetcher", githubEventsFetcher.Arn)
		ctx.Export("githubEventsConsumer", githubEventsConsumer.Arn)
		ctx.Export("sqsQueueUrl", github_event_consumer_sqs.Url)
		ctx.Export("usersTable", actorsTable.Name)
		ctx.Export("apiEndpointURL", api.Uris.MapIndex(pulumi.String("GRAPHQL")))
		ctx.Export("apiId", api.ID().ToStringOutput())

		return nil
	})
}
