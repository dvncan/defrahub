package acp

import "context"

// AccessController is the interface every ACP implementation must satisfy.
// The stub (NoopACP) allows all operations.
// A real implementation would enforce SourceHub ReBac policies.
type AccessController interface {
	CanRead(ctx context.Context, repoID, actor, resource string) (bool, error)
	CanWrite(ctx context.Context, repoID, actor, resource string) (bool, error)
	ShareRepo(ctx context.Context, repoID, relation, actor string) error
	RevokeRepo(ctx context.Context, repoID, relation, actor string) error
	ListShares(ctx context.Context, repoID string) ([]Share, error)
}

// Share represents an (actor, relation) pair.
type Share struct {
	Actor    string
	Relation string
}

// NoopACP allows everything — used until real ACP is wired.
type NoopACP struct{}

func (n *NoopACP) CanRead(_ context.Context, _, _, _ string) (bool, error)  { return true, nil }
func (n *NoopACP) CanWrite(_ context.Context, _, _, _ string) (bool, error) { return true, nil }
func (n *NoopACP) ShareRepo(_ context.Context, _, _, _ string) error        { return nil }
func (n *NoopACP) RevokeRepo(_ context.Context, _, _, _ string) error       { return nil }
func (n *NoopACP) ListShares(_ context.Context, _ string) ([]Share, error)  { return nil, nil }
