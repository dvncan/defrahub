# DefraGit

A local-first, peer-to-peer version control CLI backed by [DefraDB](https://github.com/sourcenetwork/defradb). It mirrors the Git mental model — repos, commits, branches, diffs, merges — but stores everything in DefraDB's MerkleCRDT graph, enabling decentralized collaboration via libp2p with no central server required.

---

## Build

Requirements: Go 1.24+

```bash
cd defragit
go install .
```

Or build a local binary:

```bash
go build -o defragit .
```

---

## Quick Start

### 1. Set your identity (optional — defaults to "Unknown")

DefraGit reads author info from the global config at `~/.defragit/config.yaml`. Create it once:

```yaml
user:
  name: Alice
  email: alice@example.com
```

### 2. Initialize a repository

```bash
defragit init myrepo
```

Output:
```
Initialized DefraGit repo "myrepo"
Repo ID:  bae-abc123...
Peer ID:  12D3KooW...
Branch:   main
Data dir: ~/.defragit/myrepo/
```

Each repo gets its own embedded DefraDB node at `~/.defragit/<name>/db/`.

### 3. Stage files

```bash
defragit add myrepo README.md
defragit add myrepo src/main.go src/util.go
defragit add myrepo .         # stage everything in current directory
```

### 4. Check status

```bash
defragit status myrepo
```

```
On branch main

Changes staged for commit:
  (new file)  README.md
  (new file)  src/main.go

Untracked files:
  build/output.bin
```

### 5. Diff staged changes

```bash
defragit diff myrepo                # working dir vs staged
defragit diff myrepo --staged       # staged vs last commit
defragit diff myrepo src/main.go   # specific file
```

### 6. Commit

```bash
defragit commit myrepo -m "initial commit"
```

```
[main a1b2c3d4e] initial commit
3 files changed, 42 insertions(+), 0 deletions(-)
```

### 7. View history

```bash
defragit log myrepo
defragit log myrepo --limit 5
defragit log myrepo --branch feature-x
```

---

## Branching

```bash
defragit branch myrepo              # list all branches (* = current)
defragit branch myrepo feature-x    # create branch at current HEAD
defragit branch myrepo -d old-feat  # delete branch

defragit checkout myrepo feature-x  # switch to branch
defragit checkout myrepo -b hotfix  # create and switch
```

**Note:** DefraGit does not manage working directory files — it only tracks staged and committed content. Switch branches freely; your on-disk files are yours to manage.

---

## Merging

```bash
defragit merge myrepo feature-x
```

Three-way merge against the common ancestor. On success, creates a merge commit. On conflict, writes conflict markers to the affected blobs and prompts you to resolve manually then `commit`.

---

## Peer-to-Peer Collaboration

### Overview

DefraGit uses DefraDB's libp2p replication. Each node has a **Peer ID** (shown on `init`). Peers push their collection changes to registered replicator targets.

### Alice shares her repo with Bob

**On Alice's machine:**

```bash
# 1. Get Alice's peer address (shown on init, or check config)
cat ~/.defragit/myrepo/config.yaml | grep peer_id

# 2. Share with Bob (ACP stub — not enforced yet)
defragit share myrepo add --peer <BobPeerID> --access reader
# Prints the address Bob needs:
#   defragit remote add origin /ip4/<alice-ip>/tcp/9171/p2p/<AlicePeerID>
```

**On Bob's machine:**

```bash
# 3. Initialize a local repo with the same name
defragit init myrepo

# 4. Add Alice as a remote
defragit remote add myrepo origin /ip4/192.168.1.5/tcp/9171/p2p/12D3KooWAlice...

# 5. Pull
defragit pull myrepo
```

**On Alice's machine (to push her commits to Bob):**

```bash
defragit push myrepo origin main
```

### Remote management

```bash
defragit remote add myrepo origin /ip4/192.168.1.5/tcp/9171/p2p/12D3KooW...
defragit remote list myrepo
defragit remote remove myrepo origin
```

### Share management (ACP stub)

```bash
defragit share myrepo add --peer <peerID> --access reader
defragit share myrepo add --peer <peerID> --access writer
defragit share myrepo list
defragit share myrepo revoke --peer <peerID>
```

> **Note:** ACP (access control) is currently a no-op stub. All actors can access all repos until a real `AccessController` implementation is wired into `cmd/root.go`.

---

## Data Model

All data is stored in DefraDB collections. The commit chain is a MerkleDAG — every `DG__Commit` has a content-addressed `_docID` used as the commit hash, and a `parentCID` field linking to its parent.

| Collection | Purpose |
|---|---|
| `DG__Repo` | Repository metadata |
| `DG__Branch` | Branch refs (mutable head pointer) |
| `DG__Commit` | Immutable commit snapshots |
| `DG__Tree` | Per-commit file tree entries |
| `DG__Blob` | Deduplicated file content |
| `DG__Index` | Staging area entries |
| `DG__Remote` | Known remote peers |

---

## Config Files

**Per-repo:** `~/.defragit/<repo>/config.yaml`

```yaml
repo:
  name: myrepo
  id: bae-abc123...

user:
  name: Alice
  email: alice@example.com

db:
  path: /Users/alice/.defragit/myrepo/db
  port: 9181
  p2p_port: 9171

branch:
  default: main
  current: main

identity:
  key_path: /Users/alice/.defragit/identity.key
  peer_id: 12D3KooW...
```

**Global:** `~/.defragit/config.yaml` — stores `user.name` and `user.email` applied to all new repos.

**Identity key:** `~/.defragit/identity.key` — shared secp256k1 keypair across all repos on this machine. Generated automatically on first `init`.

---

## Commit Hashes

DefraGit uses DefraDB's content-addressed `_docID` (a `bae-...` CID) as the commit hash. The first 9 characters after `bae-` are shown as the short hash in log output. Hashes are deterministic — the same initial document content always produces the same ID.
