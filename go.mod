module github.com/myorg/workspace

go 1.27

// Scratch programs live outside ./... : they must still compile when run
// explicitly (go run ./development/<name>) but never block a commit.
ignore ./development

require (
	go.yaml.in/yaml/v3 v3.0.5
	golang.org/x/tools v0.51.0
)

require (
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
)
