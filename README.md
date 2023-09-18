# PointFive exercise

# Prerequisite

- AWS account: to deploy the backend
- Pulumi: used for IaC to create AWS resources, please setup pulumi, for more details https://www.pulumi.com/docs/clouds/aws/get-started/begin/

# Build and deploy

- Clone the repo then run ./build_deploy.sh
  - This script will build the different components and run 'pulumi up' to provision AWS resources and deploy the code

# Architecture

- githubEventsFetcher, producer lambda, triggered by eventBridge each X minutes
  - Can be triggered from AWS console 'test' without parameters
  - To configure the interval change the cronExpression in https://github.com/ahmadsheikh01/pointfive_pulumi/blob/52d2e763bcabc14555e97eebe8f23cf40161c9a6/main.go#L196C46-L196C46
  - For each event send SQS message for githubEventConsumer to be processed
- githubEventsConsumer, consumer lambda, triggered by SQS, each SQS message represents github event
  - For each event saves the relevant data in dynamoDB tables
- AppSync to allow fetching the data saved in dynamoDB with lambda resolver (API)

# Known issues

- No tests were added to the project
- Emails always empty from github API
- IaC and policies in not well defined (i.e. using same role for all lambdas)
- Querying the Repos always trigger fetching each repo starts which make it slow
- Querying the DB is done using dynamoDB scan functionality
