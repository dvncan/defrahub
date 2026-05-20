# DefraGit — Project Build Prompt

You are building **DefraGit**: a local-first, peer-to-peer version control CLI backed by [DefraDB](https://github.com/sourcenetwork/defradb). It mirrors the Git mental model (repos, commits, branches, diffs, merges) but stores everything in DefraDB's MerkleCRDT graph, enabling decentralized collaboration via libp2p peer sharing — no central server required.

---

## Project Overview

| Attribute | Value |
|---|---|
| Language | Go 1.24+ |
| Database | DefraDB v1.0.0-rc1 (embedded, not standalone) |
| CLI framework | [Cobra](https://github.com/spf13/cobra) |
| Config | YAML via [Viper](https://github.com/spf13/viper) |
| Diff | [go-diff](https://github.com/sergi/go-diff) (Myers diff) |
| Entry point | `defragit` binary |

---

## Repository Layout

```
defragit/
├── cmd/
│   ├── root.go          # cobra root, global flags, config load
│   ├── init.go          # defragit init
│   ├── add.go           # defragit add
│   ├── commit.go        # defragit commit
│   ├── status.go        # defragit status
│   ├── diff.go          # defragit diff
│   ├── log.go           # defragit log
│   ├── branch.go        # defragit branch / checkout
│   ├── merge.go         # defragit merge
│   ├── push.go          # defragit push  (save to DefraDB)
│   ├── pull.go          # defragit pull  (fetch from peer)
│   ├── remote.go        # defragit remote add/list/remove
│   └── share.go         # defragit share (ACP stub)
├── internal/
│   ├── db/
│   │   ├── node.go      # DefraDB node lifecycle
│   │   ├── schema.go    # schema registration helpers
│   │   └── query.go     # typed GQL query helpers
│   ├── repo/
│   │   ├── repo.go      # Repo struct, open/create
│   │   ├── index.go     # staging area (index)
│   │   ├── object.go    # file object (blob) store
│   │   ├── commit.go    # commit creation / traversal
│   │   ├── branch.go    # branch / HEAD management
│   │   ├── diff.go      # diff engine
│   │   └── merge.go     # three-way merge
│   ├── p2p/
│   │   ├── peer.go      # peer connection helpers (wraps DefraDB libp2p)
│   │   └── sync.go      # replicator registration
│   └── acp/
│       └── stub.go      # ACP interface + no-op implementation
├── config/
│   └── config.go        # config struct, defaults
├── .defragit/           # per-repo runtime dir (gitignored from DefraDB data)
│   ├── config.yaml
│   └── HEAD
├── go.mod
├── go.sum
└── README.md
```

---

## DefraDB Schema Design

All DefraDB collections are registered once on `defragit init`. Use double-underscore (`__`) for namespacing per Shinzo convention.

### Important Rules (from DefraDB reference)
- **Never** define `_docID`, `_version`, `_deleted` in SDL — they are auto-generated.
- Use `SaveDocument` / `SaveManyDocuments` (not `Create`/`CreateMany` — removed in v0.20+).
- Use `AddCollection` (not `AddSchema` — removed in v0.20+).
- Use `PatchCollection` for schema evolution — never re-add an existing type.
- `ExecRequest` returns `result.GQL.Data` as `any`; cast via `map[string]any` → `[]any`.

### Collections

```graphql
# A tracked repository
type DG__Repo {
  name:        String
  description: String
  createdAt:   DateTime
  ownerPeerID: String
}

# A branch ref — points at a commit hash (content ID)
type DG__Branch {
  repoID:    String   # _docID of DG__Repo
  name:      String
  headCID:   String   # CID of the tip DG__Commit
  isDefault: Boolean
}

# An immutable commit snapshot
type DG__Commit {
  repoID:     String
  branchName: String
  parentCID:  String   # empty string = root commit
  message:    String
  authorName: String
  authorEmail: String
  timestamp:  DateTime
  treeCID:    String   # CID of the root DG__Tree
}

# A tree node (directory snapshot)
type DG__Tree {
  commitCID: String
  path:      String   # relative path, e.g. "src/main.go"
  blobCID:   String   # content hash
  mode:      String   # "file" | "dir"
}

# File content (blob), deduplicated by content hash
type DG__Blob {
  contentHash: String   # sha256 hex of content
  content:     String   # raw file content (UTF-8; base64 for binary)
  size:        Int
  encoding:    String   # "utf8" | "base64"
}

# Staging index entry
type DG__Index {
  repoID:      String
  filePath:    String
  blobCID:     String
  contentHash: String
  stagedAt:    DateTime
}

# Known remote peers
type DG__Remote {
  repoID:   String
  name:     String   # e.g. "origin"
  peerID:   String   # libp2p peer ID multiaddr
  peerAddr: String   # full multiaddr including /p2p/...
}
```

---

## ACP Stub Design

Build a clean interface in `internal/acp/stub.go` so ACP can be swapped in later without touching business logic.

```go
package acp

import "context"

// AccessController is the interface every ACP implementation must satisfy.
// The stub (NoopACP) allows all operations.
// A real implementation would enforce SourceHub ReBac policies.
type AccessController interface {
    // CanRead returns true if actor may read the named resource in the repo.
    CanRead(ctx context.Context, repoID, actor, resource string) (bool, error)

    // CanWrite returns true if actor may write to the named resource.
    CanWrite(ctx context.Context, repoID, actor, resource string) (bool, error)

    // ShareRepo grants relation to actor for the entire repo.
    // relation is one of: "reader" | "writer" | "owner"
    ShareRepo(ctx context.Context, repoID, relation, actor string) error

    // RevokeRepo removes relation from actor.
    RevokeRepo(ctx context.Context, repoID, relation, actor string) error

    // ListShares returns all (actor, relation) pairs for the repo.
    ListShares(ctx context.Context, repoID string) ([]Share, error)
}

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
```

**Future ACP implementation note (do not implement now — only stub):**
- On `defragit init`, add a DefraDB policy (`DG__Repo` resource, relations: `owner`, `reader`, `writer`)
- Default: block everything unless explicitly shared
- Policy expression for `read`: `owner + reader + writer`
- Policy expression for `write`: `owner + writer`
- `owner` relation set to the initializing peer's DID
- `ShareRepo` maps to `defradb client acp relationship add`

---

## Peer Sharing Scheme

### Peer Identity

On first `defragit init`, generate or load a `secp256k1` keypair stored in `~/.defragit/identity.key`. The peer ID is derived from this key (libp2p standard). Display it as:

```
Your Peer ID: 12D3KooW...
```

### Sharing a Repo with Another Peer

```bash
# Owner shares their repo with a collaborator by peer ID
defragit share add --repo myrepo --peer 12D3KooWXyz... --access reader

# List who has access
defragit share list --repo myrepo

# Revoke access
defragit share revoke --repo myrepo --peer 12D3KooWXyz...
```

Implementation (stub phase):
1. `share add` calls `acp.ShareRepo(repoID, "reader", peerID)` — no-op for now.
2. Prints a connection string the collaborator uses to pull:
   ```
   defragit remote add origin /ip4/192.168.1.5/tcp/9171/p2p/12D3KooW...
   ```
3. On `pull`, DefraDB's P2P replicator is used to sync the relevant collections.

### Collaboration Flow (design intent, implement fully)

```
Alice (owner)                           Bob (collaborator)
─────────────────────────────────────────────────────────
defragit init myrepo
defragit share add --peer <BobPeerID> --access reader
  → prints: /ip4/.../p2p/<AlicePeerID>
                                        defragit remote add origin <AlicePeerAddr>
                                        defragit pull origin myrepo
                                          → uses DefraDB replicator to sync
                                        defragit log  (sees Alice's commits)
```

---

## CLI Commands — Full Specification

### `defragit init [name]`

- Creates `~/.defragit/<name>/` directory.
- Starts an embedded DefraDB node at `~/.defragit/<name>/db/`.
- Registers all schemas via `AddCollection`.
- Creates a `DG__Repo` document, saves the `_docID` to config.
- Creates default branch `main` with empty `headCID`.
- Writes `~/.defragit/<name>/config.yaml` with repo metadata and defradb path.
- Calls `acp.ShareRepo` (no-op) to set up ownership stub.
- Prints peer ID and repo ID.

```
Initialized DefraGit repo 'myrepo'
Repo ID:   bae-abc123...
Peer ID:   12D3KooW...
Branch:    main
Data dir:  ~/.defragit/myrepo/
```

### `defragit add <file> [file...]` / `defragit add .`

- Reads each file from disk.
- Computes `sha256` content hash.
- Checks if a `DG__Blob` with that `contentHash` already exists (query by filter). If not, saves it.
- Upserts a `DG__Index` entry (one per file path, replace if exists).
- Prints: `staged: src/main.go`

### `defragit status`

- Loads all `DG__Index` entries for the current repo.
- For each staged file, compares contentHash against the last committed `DG__Tree` entry for the current branch.
- Scans working directory for untracked files (not in index or last tree).
- Prints:
  ```
  On branch main
  
  Changes staged for commit:
    (new file)  src/main.go
    (modified)  README.md
  
  Untracked files:
    build/output.bin
  ```

### `defragit diff [--staged] [file]`

- Without `--staged`: diffs working directory file against the staged `DG__Blob`.
- With `--staged`: diffs staged `DG__Blob` against last committed blob for that path.
- Uses `go-diff` Myers diff, output in unified diff format with +/- lines and context.
- If no file given, diffs all changed files.

```diff
--- a/src/main.go
+++ b/src/main.go
@@ -1,5 +1,7 @@
 package main
 
+import "fmt"
+
 func main() {
-    println("hello")
+    fmt.Println("hello, defragit")
 }
```

### `defragit commit -m "message"`

1. Load all `DG__Index` entries for the repo.
2. For each entry, write a `DG__Tree` document (`commitCID` will be filled after commit is created — use a two-phase approach or store treeCID after).
3. Create a `DG__Commit` document with:
   - `parentCID`: current branch `headCID` (empty if root commit)
   - `message`, `authorName`, `authorEmail` from config
   - `timestamp`: now
   - `treeCID`: composite identifier linking to this commit's tree entries
4. Update the `DG__Branch` document's `headCID` to the new commit's `_docID`.
5. Clear all `DG__Index` entries for the repo (staged → committed).
6. Print:
   ```
   [main a1b2c3d] add initial files
   3 files changed, 42 insertions(+), 0 deletions(-)
   ```

**Note on CIDs:** DefraDB assigns content-addressed `_docID` (a `bae-...` CID). Use this as the "commit hash". Display the first 7 chars as the short hash in output.

### `defragit log [--branch <name>] [--limit N]`

- Loads the branch's `headCID`.
- Traverses commit chain via `parentCID` links until `parentCID` is empty.
- Prints each commit:
  ```
  commit bae-a1b2c3d...
  Author: Alice <alice@example.com>
  Date:   2026-05-19 14:32:00
  
      add initial files
  ```

### `defragit branch`

Subcommands:

```bash
defragit branch              # list all branches (* marks current)
defragit branch <name>       # create branch at current HEAD
defragit branch -d <name>    # delete branch
defragit checkout <name>     # switch HEAD to branch
defragit checkout -b <name>  # create and switch
```

- Branch state stored in `DG__Branch` collection.
- Current branch stored in `~/.defragit/<repo>/HEAD` file (plain text: `ref: refs/heads/main`).
- `checkout` updates HEAD file; does NOT touch working directory (like git's index — working files are user-managed).

### `defragit merge <branch>`

Three-way merge:

1. Find common ancestor commit (walk both branch histories, find first shared `_docID`).
2. For each file in the merged branch's tree: compare base → ours and base → theirs.
3. If only one side changed: take that side.
4. If both changed identically: take either.
5. If both changed differently: mark as conflict.
6. Write conflict files with markers:
   ```
   <<<<<<< HEAD
   our version
   =======
   their version
   >>>>>>> feature-branch
   ```
7. Auto-merge creates a new commit if no conflicts.
8. If conflicts: print files with conflicts, leave user to resolve, require `defragit commit` after.

### `defragit push [remote] [branch]`

- Default remote: `origin`, default branch: current HEAD.
- Registers a DefraDB replicator for the relevant collections pointing at the remote peer addr.
- Uses `defraNode` P2P replicator API to push `DG__Commit`, `DG__Tree`, `DG__Blob`, `DG__Branch` for the repo.
- Prints:
  ```
  Pushing main → origin (12D3KooW...)
  ✓ 3 commits synced
  ```

Implementation via DefraDB Go embedded:
```go
// Register replicator for each collection
err := defraNode.Peer.SetReplicator(ctx, client.Replicator{
    Info:        peerInfo,  // peer.AddrInfo
    Schemas:     []string{"DG__Commit", "DG__Tree", "DG__Blob", "DG__Branch", "DG__Repo"},
})
```

### `defragit pull [remote] [branch]`

- Resolves remote peer addr from `DG__Remote`.
- Connects to the peer via DefraDB libp2p (pubsub subscription).
- Fetches updated `DG__Branch` head for the named branch.
- Walks new commits and blobs not present locally.
- Applies a fast-forward merge if local branch has no divergent commits; otherwise prompts user to merge.
- Prints:
  ```
  Pulling main ← origin (12D3KooW...)
  ✓ 2 new commits
  Fast-forward to bae-d4e5f6...
  ```

### `defragit remote`

```bash
defragit remote add <name> <peerAddr>    # store DG__Remote
defragit remote list                     # print all remotes
defragit remote remove <name>            # delete DG__Remote
```

### `defragit share`

```bash
defragit share add --peer <peerID> --access reader|writer
defragit share list
defragit share revoke --peer <peerID>
```

- Currently routes through `NoopACP`.
- Prints a reminder message:
  ```
  [ACP stub] Access control not enforced yet.
  Share this address with your collaborator:
    defragit remote add origin /ip4/<your-ip>/tcp/9171/p2p/<your-peerID>
  ```

---

## Config File Format

`~/.defragit/<repo>/config.yaml`:

```yaml
repo:
  name: myrepo
  id: bae-abc123...          # DG__Repo _docID
  description: ""

user:
  name: Alice
  email: alice@example.com

db:
  path: ~/.defragit/myrepo/db
  port: 9181
  p2pPort: 9171

branch:
  default: main

identity:
  keyPath: ~/.defragit/identity.key   # shared across all repos on this machine
  peerID: 12D3KooW...
```

Global config at `~/.defragit/config.yaml` stores `user.name`, `user.email`, `identity.*`.

---

## DefraDB Node Lifecycle

Every command that needs the database:

```go
func openNode(ctx context.Context, cfg *config.Config) (*node.Node, error) {
    n, err := node.New(ctx,
        node.WithStorePath(cfg.DB.Path),
        // P2P is started only for push/pull/share commands
        // For read/write commands, P2P can be disabled to keep startup fast
    )
    if err != nil {
        return nil, err
    }
    return n, n.Start(ctx)
}
```

Shut down cleanly in every command's `RunE`:

```go
defer func() {
    if err := defraNode.Close(ctx); err != nil {
        log.Warn("error closing DefraDB node", err)
    }
}()
```

---

## Error Handling Conventions

- All errors bubble up to Cobra's `RunE` and are printed by root command.
- Wrap errors with context: `fmt.Errorf("commit: writing tree: %w", err)`.
- DefraDB `ExecRequest` errors live in `result.GQL.Errors` (a slice) — always check before using `result.GQL.Data`.
- Schema-not-found errors on startup → trigger `ensureSchema()` idempotent registration.

---

## Data Casting Pattern (DefraDB)

Use this pattern everywhere you query DefraDB:

```go
func extractDocs(result client.RequestResult, typeName string) ([]map[string]any, error) {
    if result.GQL.Errors != nil {
        return nil, fmt.Errorf("gql error: %v", result.GQL.Errors)``
    }
    data, ok := result.GQL.Data.(map[string]any)
    if !ok {
        return nil, fmt.Errorf("unexpected data shape")
    }
    raw, ok := data[typeName].([]any)
    if !ok {
        return nil, nil // no results
    }
    docs := make([]map[string]any, 0, len(raw))
    for _, r := range raw {
        if m, ok := r.(map[string]any); ok {
            docs = append(docs, m)
        }
    }
    return docs, nil
}
```

---

## Schema Registration (Idempotent)

On every command startup, call `ensureSchema`. Use a version sentinel stored in a `DG__SchemaVersion` collection to skip re-registration if already done:

```go
type DG__SchemaVersion {
  version: Int
  appliedAt: DateTime
}
```

If version matches current, skip. If missing or old, call `AddCollection` for each new type, or `PatchCollection` for schema changes.

---

## Output Formatting

- Use ANSI color codes directly (no heavy dependency):
  - Green `\033[32m` for additions/success
  - Red `\033[31m` for deletions/errors
  - Yellow `\033[33m` for warnings/untracked
  - Cyan `\033[36m` for branch names / commit hashes
  - Bold `\033[1m` for headers
  - Reset `\033[0m`
- Column-align output like `git status` does.
- Short commit hash = first 9 chars of `_docID` after `bae-` prefix.

---

## Testing Strategy

- Unit tests for `internal/repo/` with an in-memory DefraDB node (use `node.WithStorePath("")` for temp).
- Table-driven tests for diff and merge logic.
- Integration test: `TestFullWorkflow` — init, add, commit, branch, checkout, commit, merge.
- Mock `AccessController` for ACP tests.

---

## Implementation Order

Build in this sequence:

1. **`internal/db/`** — DefraDB node lifecycle, schema registration, query helpers
2. **`internal/acp/stub.go`** — NoopACP
3. **`config/`** — config load/save
4. **`cmd/root.go`** — Cobra root, global setup
5. **`defragit init`** + `internal/repo/repo.go`
6. **`defragit add`** + `internal/repo/object.go` (blob store)
7. **`defragit status`** + `internal/repo/index.go`
8. **`defragit diff`** + `internal/repo/diff.go`
9. **`defragit commit`** + `internal/repo/commit.go`
10. **`defragit log`**
11. **`defragit branch`** + `defragit checkout`
12. **`defragit merge`** + `internal/repo/merge.go`
13. **`defragit remote`**
14. **`defragit push`** + **`defragit pull`** + `internal/p2p/`
15. **`defragit share`** (ACP stub wiring)
16. Tests, README

---

## Key Invariants to Maintain

1. `_docID` is your commit hash. Never generate your own hash for commits — let DefraDB's MerkleCRDT assign it.
2. Blobs are **content-addressed and deduplicated** — query by `contentHash` before writing a new blob.
3. Index entries are **per-repo** — always filter by `repoID`.
4. Branch `headCID` is the only mutable pointer in the system. Everything else is append-only.
5. The working directory is **not managed** by DefraGit — only staged/committed content lives in DefraDB. Users edit files normally on disk.
6. P2P is **opt-in** — most commands work without starting the p2p stack (faster startup for local-only workflows).
7. ACP interface is **injected** — commands receive an `AccessController` from the root setup. Swapping NoopACP for a real implementation requires changing one line in `root.go`.

---

## Future ACP Implementation Notes (do not build now)

When ACP is wired in:

- Add policy registration to `defragit init` using the DefraDB policy SDL described in the reference.
- `DPI compliance`: `read: owner + reader + writer`, `write: owner + writer`.
- Owner = initializing peer's DID (`did:key:<secp256k1-pubkey>`).
- `defragit share add` calls real `acp.relationship add` via DefraDB client.
- P2P nodes enforce access via the SourceHub ACP mode (not local ACP — local ACP doesn't work cross-peer).
- Default: **block all** — no relation = no access. This is already the DefraDB default when a policy is attached.
- Token generation: `FullIdentity.NewToken(duration, audience, authorizedAccount)` for HTTP bearer auth.

---

*Build this project as a clean, idiomatic Go CLI. Prioritize correctness of the MerkleDAG commit chain and blob deduplication. The ACP seam must be clean — a future developer should be able to implement `AccessController` without reading business logic.*
