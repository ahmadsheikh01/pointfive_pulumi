package common

var Version string = "1.0"

type Github_event struct {
	ActorName  string
	ActorLogin string
	ActorEmail string
	RepoUrl    string
	RepoName   string
	RepoId     int64
	EventType  string
}
