// Package component provides component templates used by the erato web app.
package component

import "google.golang.org/protobuf/types/known/timestamppb"

// Resource represents the common interface between an Entry and Chapter.
type Resource interface {
	GetPath() string
	GetDisplayName() string
	GetUpdateTime() *timestamppb.Timestamp
	HasViewTime() bool
	GetViewTime() *timestamppb.Timestamp
	HasReadTime() bool
	GetReadTime() *timestamppb.Timestamp
}
