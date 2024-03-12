package awsfs

import (
	"path"
	"time"
)

type Server struct {
	MetadataStore MetadataStore
	PhysicalStore PhysicalStore
}

// slashClean is equivalent to but slightly more efficient than
// path.Clean("/" + name).
func slashClean(name string) string {
	if name == "" || name[0] != '/' {
		name = "/" + name
	}
	return path.Clean(name)
}

type EntryType string

const (
	EntryTypeFile EntryType = "file"
	EntryTypeDir  EntryType = "dir"
)

type Reference struct {
	ID      string            `dynamodbav:"id"`
	Entries map[string]string `dynamodbav:"entries"`
	Version int               `dynamodbav:"version"`
}

type Entry struct {
	ID       string    `dynamodbav:"id"`
	ParentID string    `dynamodbav:"parent_id"`
	Name     string    `dynamodbav:"name"`
	Type     EntryType `dynamodbav:"type"`
	Size     int64     `dynamodbav:"size"`
	Modify   time.Time `dynamodbav:"modify"`
	Version  int       `dynamodbav:"version"`
}
