package instance

import (
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
)

// ListServerRequest is a convenience type for the Scaleway `ListServerRequest` API request
type ListServerRequest instance.ListServersRequest

// Native returns the native `*instance.ListServersRequest` type
func (l *ListServerRequest) Native() *instance.ListServersRequest {
	return (*instance.ListServersRequest)(l)
}

// Next increments the page field and returns a new `ListServerRequest`
func (l ListServerRequest) Next() *ListServerRequest {
	(*l.Page)++
	return &l
}
