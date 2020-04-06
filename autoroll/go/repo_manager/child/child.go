package child

/*
   Package child contains implementations of the Child interface.
*/

import (
	"context"

	"go.skia.org/infra/autoroll/go/revision"
)

// Child represents a Child (git repo or otherwise) which can be rolled into a
// Parent.
type Child interface {
	// Update updates the local view of the Child and returns the tip-of-
	// tree Revision and the list of not-yet-rolled revisions, or any error
	// which occurred, given the last-rolled revision.
	Update(context.Context, *revision.Revision) (*revision.Revision, []*revision.Revision, error)

	// GetRevision returns a Revision instance associated with the given
	// revision ID.
	GetRevision(context.Context, string) (*revision.Revision, error)

	// Download downloads the Child at the given Revision to the given
	// destination.
	Download(context.Context, *revision.Revision, string) error
}
