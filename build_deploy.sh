
rm -rf ./tmp

cd githubEventsFetcher
GOOS=linux go build -o ../tmp/githubEventsFetcher githubEventsFetcher.go
cd ..
zip -j ./tmp/githubEventsFetcher.zip ./tmp/githubEventsFetcher

cd githubEventsConsumer && GOOS=linux go build -o ../tmp/githubEventsConsumer githubEventsConsumer.go
cd ..
zip -j ./tmp/githubEventsConsumer.zip ./tmp/githubEventsConsumer

cd API && GOOS=linux go build -o ../tmp/api api.go
cd ..
zip -j ./tmp/api.zip ./tmp/api

pulumi up --yes