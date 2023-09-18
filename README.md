# PointFive exercise

# Prerequisite

- AWS account: to deploy the backend
- Pulumi: used for IaC to create AWS resources, please set up pulumi, for more details https://www.pulumi.com/docs/clouds/aws/get-started/begin/

# Build and deploy

- Clone the repo then run ./build_deploy.sh
  - This script will build the different components and run 'pulumi up' to provision AWS resources and deploy the code
- To remove resources from aws run 'pulumi destroy'

# Architecture

- githubEventsFetcher, producer lambda, triggered by eventBridge each X minutes
  - Can be triggered from AWS console 'test' without parameters
  - To configure the interval change the cron expression in https://github.com/ahmadsheikh01/pointfive_pulumi/blob/52d2e763bcabc14555e97eebe8f23cf40161c9a6/main.go#L196C46-L196C46
  - For each event send SQS message to githubEventConsumer to be processed
- githubEventsConsumer, consumer lambda, triggered by SQS, each SQS message represents github event
  - For each event save the relevant data in dynamoDB tables
- AppSync to allow fetching the data saved in dynamoDB with lambda resolver (API)

# Known issues

- No tests were added to the project
- Emails always empty from github API
- IaC and policies is not well defined (i.e. using the same role for all lambdas)
- Querying the Repos always triggers fetching each repo starts which makes it slow
- Querying the DB is done using dynamoDB scan functionality
