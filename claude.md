# DefraDB — Claude Reference Guide

> **Source:** https://github.com/sourcenetwork/defradb  
> **Docs:** https://docs.source.network/defradb  
> **Latest stable release:** v1.0.0-rc1 (March 2026)  
> **Language:** Go 1.24+  
> **License:** Business Source License (BSL 1.1) — turns Apache 2.0 after 4 years

---

## What Is DefraDB?

DefraDB is a **peer-to-peer, edge-first, document database** built on:
- **MerkleCRDTs** — each document has an immutable, append-only commit graph (like Git)
- **IPLD** — content-addressable data (every document/commit has a CID)
- **libp2p** — decentralized networking and replication
- **DQL** — GraphQL-compatible query language with extensions

It is the core storage engine used in the **Shinzo Network** (indexer client, host client) and powers the Source ecosystem.

---

## Architecture Overview

```
Application / Claude
       |
       v
  GraphQL API (:9181/api/graphql or :9181/api/v1/graphql)
       |
       v
   DefraDB Node
   ┌─────────────────────────┐
   │  Collections (schemas)  │
   │  MerkleDAG (commits)    │
   │  ACP (access control)   │
   │  libp2p (P2P sync)      │
   └─────────────────────────┘
       |
       v
  BadgerDB (local storage)
```

---

## Default Ports & Endpoints

| Endpoint | Default |
|---|---|
| GraphQL API | `http://localhost:9181/api/graphql` |
| Versioned API | `http://localhost:9181/api/v1/graphql` |
| P2P networking | `localhost:9171` |
| GraphQL Playground | `:9182` (dev/playground build) |

Expose externally with:
```bash
defradb start --p2paddr /ip4/0.0.0.0/tcp/9171 --url 0.0.0.0:9181
```

---

## Installation & Start

```bash
git clone https://github.com/sourcenetwork/defradb.git
cd defradb
make install
export PATH=$PATH:$(go env GOPATH)/bin

# Start a node
defradb start
```

Or via Docker (Shinzo context):
```yaml
environment:
  - DEFRA_URL=0.0.0.0:9181
  - GOMEMLIMIT=14GiB
ports:
  - "9181:9181"
  - "9171:9171"
```

---

## Build Playground

```bash
cd defra 
make deps:playground
GOFLAGS="-tags=playground" make build

# Start a node
./build/defradb start 

# Open a browser
open http://localhost:9182
```

---

## Key Management

DefraDB uses a built-in keyring. Keys are randomly generated on first start.

```bash
# Generate new keys
defradb keyring new

# Import existing key
defradb keyring add <name> <private-key-hex>
```

Required keys:
- `peer-key` — Ed25519 (required, for P2P)
- `encryption-key` — AES key (optional)
- `node-identity-key` — Secp256k1 (optional, for node identity)

Must set `DEFRA_KEYRING_SECRET` env var (or `.env` file in working dir) to unlock keyring on start.

---

## Schema / Collections

### IMPORTANT: System-Generated Fields — Do NOT Include in Schema

DefraDB **automatically generates** these fields. Never define them in SDL:

| Field | Description |
|---|---|
| `_docID` | Unique document identifier (content-addressed) |
| `_version` | MerkleDAG version/height |
| `_deleted` | Soft-delete flag |
| `_key` | Internal key (older versions) |

They ARE queryable and returnable, but defining them in your SDL will cause errors.

### SDL Schema Definition

DefraDB uses **GraphQL SDL** (Schema Definition Language) for collection schemas.

```graphql
type User {
  name: String
  age: Int
  verified: Boolean
  points: Float
}
```

Supported scalar types: `String`, `Int`, `Float`, `Boolean`, `ID`, `DateTime`, `Blob`, `JSON`

### Adding a Collection via CLI

```bash
defradb client collection add '
  type User {
    name: String
    age: Int
    verified: Boolean
    points: Float
  }
'
```

### Adding a Collection via Go (Embedded — v0.20+)

**Note:** `AddSchema` was removed. Use `AddCollection`:

```go
_, err := defraNode.DB.AddCollection(ctx, sdlString)
```

Signature:
```go
AddCollection(
    ctx context.Context,
    sdl string,
    opts ...options.Enumerable[options.AddCollectionOptions],
) ([]CollectionVersion, error)
```

### Patching Collections (Schema Updates)

```go
err := defraNode.DB.PatchCollection(ctx, jsonPatchString)
```

---

## CRUD Operations

### GraphQL Mutations (HTTP or CLI)

**Create:**
```graphql
mutation {
  add_User(input: {age: 31, verified: true, points: 90, name: "Bob"}) {
    _docID
  }
}
```

**Update by docID:**
```graphql
mutation {
  update_User(docID: "bae-91171025-...", input: {age: 32}) {
    _docID
    age
  }
}
```

**Delete:**
```graphql
mutation {
  delete_User(docID: "bae-91171025-...") {
    _docID
  }
}
```

### Go (Embedded) — v0.20+

`Create` and `CreateMany` were removed. Use:

```go
// Upsert (create or update by docID)
err := collection.SaveDocument(ctx, doc)

// Bulk upsert
err := collection.SaveManyDocuments(ctx, docs)
```

Signatures:
```go
SaveDocument(ctx context.Context, doc *Document, opts ...options.Enumerable[options.SaveDocumentOptions]) error
SaveManyDocuments(ctx context.Context, docs []*Document, opts ...options.Enumerable[options.SaveDocumentOptions]) error
```

---

## Querying

### Basic Query

```graphql
query {
  User {
    _docID
    name
    age
    points
  }
}
```

GraphQL only returns fields you explicitly request — there is no `SELECT *`.

### Filtering

```graphql
query {
  User(filter: {points: {_geq: 50}}) {
    _docID
    name
    points
  }
}
```

Filter operators:
| Operator | Meaning |
|---|---|
| `_eq` | equals |
| `_ne` | not equals |
| `_gt` | greater than |
| `_geq` | greater than or equal |
| `_lt` | less than |
| `_leq` | less than or equal |
| `_in` | in list |
| `_nin` | not in list |
| `_like` | LIKE pattern |
| `_ilike` | case-insensitive LIKE |
| `_and` | logical AND |
| `_or` | logical OR |
| `_not` | logical NOT |

### Sorting & Ordering

```graphql
query {
  User(order: {age: ASC}) {
    name
    age
  }
}
```

### Limiting & Pagination

```graphql
query {
  User(limit: 10, offset: 20) {
    name
  }
}
```

### Aggregate Functions

```graphql
query {
  _count(User: {})
}
```

```graphql
query {
  User {
    _group {
      verified
      _count
    }
  }
}
```

### Relationships (Type Joins)

Define in SDL:
```graphql
type Author {
  name: String
  articles: [Article]
}

type Article {
  title: String
  author: Author
}
```

Query:
```graphql
query {
  Author {
    name
    articles {
      title
    }
  }
}
```

### Query via Go (ExecRequest)

```go
result := defraNode.DB.ExecRequest(ctx, `
  query {
    User(filter: {age: {_geq: 18}}) {
      _docID
      name
      age
    }
  }
`)
if result.GQL.Errors != nil {
  // handle errors
}
data := result.GQL.Data
```

Pattern to extract results:
```go
if data, ok := result.GQL.Data.(map[string]any); ok {
  if users, ok := data["User"].([]any); ok {
    for _, u := range users {
      if user, ok := u.(map[string]any); ok {
        docID := user["_docID"].(string)
        // etc.
      }
    }
  }
}
```

---

## MerkleDAG — Document Commits

Every document update creates an immutable commit. Query commit history:

```graphql
query {
  _commits(docID: "bae-91171025-...") {
    cid
    delta
    height
    links {
      cid
      name
    }
  }
}
```

Get a specific commit by CID:
```graphql
query {
  _commits(cid: "bafybeif...") {
    cid
    delta
    height
  }
}
```

---

## HTTP API Authentication (Bearer Token / JWT)

The HTTP API uses **JWT bearer tokens** signed with a `secp256k1` private key.

### Token Requirements

JWT must include:
- `sub` — public key of the identity
- `aud` — hostname of the DefraDB API (e.g. `"localhost:9181"`)
- `exp` — expiry (short-lived recommended)
- `nbf` — not-before

For SourceHub ACP also set:
- `iss` — user's DID (e.g. `"did:key:z6Mk..."`)
- `iat` — current unix timestamp
- `authorized_account` — SourceHub address signing on your behalf

### Setting the Header

```
Authorization: bearer <signed-jwt>
```

A `403 Forbidden` is returned on auth failure.

### Generate secp256k1 Private Key

```bash
openssl ecparam -name secp256k1 -genkey | openssl ec -text -noout | head -n5 | tail -n3 | tr -d '\n:\ '
# Output: e3b722906ee4e56368f581cd8b18ab0f48af1ea53e635e3f7b8acd076676f6ac
```

### CLI: Use Identity Flag

```bash
defradb client collection create --name Users '[{"name": "Alice"}]' \
  --identity e3b722906ee4e56368f581cd8b18ab0f48af1ea53e635e3f7b8acd076676f6ac
```

### Go (Embedded): Identity Context

Identity management moved to `internal/identity` (not importable externally). In embedded Go usage within the Shinzo indexer, identity is injected via the `FullIdentity` interface:

```go
// FullIdentity interface (acp/identity package)
type FullIdentity interface {
    TokenIdentity
    PrivateKey() crypto.PrivateKey
    IntoRawIdentity() RawIdentity
    NewToken(duration time.Duration, audience immutable.Option[string], authorizedAccount immutable.Option[string]) ([]byte, error)
    SetBearerToken(token string)
    UpdateToken(duration time.Duration, audience immutable.Option[string], authorizedAccount immutable.Option[string]) error
}
```

The `BearerToken()` from a `TokenIdentity` is the signed JWT to use in HTTP `Authorization: bearer <token>` headers.

---

## Access Control System (ACP)

DefraDB uses **Relation-Based Access Control (ReBac)**, similar to Google Zanzibar, powered by SourceHub.

### Concepts

- **Policy** — YAML document defining resources, relations, and permissions
- **Resource** — corresponds to a collection type
- **Relation** — `owner`, `reader`, `writer`, etc.
- **Actor** — an identity (DID)
- **Object** — a document

### DPI (DefraDB Policy Interface) Rules

For a policy resource to be DPI-compliant:
1. Must have an `owner` relation
2. Required permissions (`read`, `write`) must reference `owner` first in expression
3. `owner` must be the first in any `expr` using set union (`+`)

Valid expressions:
```
expr: owner
expr: owner + reader
expr: owner + reader + writer
```

Invalid (will fail DPI):
```
expr: reader + owner    ← owner not first
expr: owner & reader    ← wrong operator
```

### Add a Policy

```yaml
# policy.yml
description: My Policy

actor:
  name: actor

resources:
  users:
    permissions:
      read:
        expr: owner + reader
      write:
        expr: owner
    relations:
      owner:
        types:
          - actor
      reader:
        types:
          - actor
```

```bash
defradb client acp policy add -f policy.yml \
  --identity <private-key-hex>
# Returns: { "PolicyID": "50d354a..." }
```

### Link Policy to Collection

```graphql
type Users @policy(
  id: "50d354a91ab1b8fce8a0ae4693de7616fb1d82cfc540f25cfbe11eb0195a5765",
  resource: "users"
) {
  name: String
  age: Int
}
```

```bash
defradb client collection add -f schema.graphql
```

### Share Document with Another Actor

```bash
defradb client acp relationship add \
  --collection Users \
  --docID bae-ff3ceb1c-... \
  --relation reader \
  --actor did:key:z7r8os2G88... \
  --identity <owner-private-key-hex>
```

Use `--actor "*"` to grant access to all actors (public).

### Revoke Access

```bash
defradb client acp relationship delete \
  --collection Users \
  --docID bae-ff3ceb1c-... \
  --relation reader \
  --actor did:key:z7r8os2G88... \
  --identity <owner-private-key-hex>
```

---

## Peer-to-Peer (P2P)

### Pubsub Peering (passive sync)

Both nodes must already have the document. Updates are broadcast on commit.

```bash
defradb start --peers /ip4/192.168.1.12/tcp/9171/p2p/<peerID>
```

### Replicator Peering (active push)

NodeA actively pushes a collection's changes to NodeB.

```bash
defradb client p2p replicator add -c Article <nodeB-peer-info-json>
```

### Subscribe to Collection Updates

```bash
defradb client p2p collection add --url localhost:9182 <collectionID>
```

### Get Peer Info

```bash
defradb client p2p info
```

---

## Schema Introspection

```graphql
query {
  _schema {
    collections {
      name
    }
  }
}
```

---

## Backup & Restore

```bash
# Export current state (JSON, no history)
defradb client backup export path/to/backup.json
defradb client backup export --pretty path/to/backup.json

# Import/restore
defradb client backup import path/to/backup.json
```

Note: Backup does not include full MerkleDAG history — only current document state.

---

## CORS Configuration

```bash
defradb start --allowed-origins=https://yourdomain.com
# or
defradb start --allowed-origins=http://localhost:3000
# or wildcard
defradb start --allowed-origins=*
```

---

## TLS Configuration

```bash
# With auto self-signed certs in ~/.defradb/certs/
defradb start --tls

# With custom cert paths
defradb start --tls --pubkeypath ~/path-to-pubkey.key --privkeypath ~/path-to-privkey.crt
```

---

## Go Embedded Usage (Shinzo Indexer Pattern)

### Import

```go
import (
    "github.com/sourcenetwork/defradb/node"
    "github.com/sourcenetwork/defradb/client"
)
```

### Start Node

```go
defraNode, err := node.New(ctx, node.WithStorePath(dataPath))
if err != nil { ... }
err = defraNode.Start(ctx)
```

### Add Collection

```go
_, err := defraNode.DB.AddCollection(ctx, `
  type Block {
    blockNumber: Int
    hash: String
    timestamp: DateTime
  }
`)
```

### Get Collection

```go
col, err := defraNode.DB.GetCollectionByName(ctx, "Block")
```

### Save a Document

```go
doc, err := client.NewDocFromJSON([]byte(`{"blockNumber": 1234, "hash": "0xabc..."}`), col.Schema())
err = col.SaveDocument(ctx, doc)
docID := doc.ID().String() // returns "bae-..."
```

### Execute GraphQL (read/write)

```go
result := defraNode.DB.ExecRequest(ctx, `mutation {
  add_Block(input: {blockNumber: 1234, hash: "0xabc"}) {
    _docID
  }
}`)
if result.GQL.Errors != nil { ... }
```

### Update via Mutation

```go
mutation := fmt.Sprintf(`mutation {
  update_Config(docID: "%s", input: {page: %d}) {
    _docID
  }
}`, docID, page)
result := defraNode.DB.ExecRequest(ctx, mutation)
```

---

## Common Gotchas & Learned Behaviours

### 1. System fields are auto-generated — never define in SDL
`_docID`, `_version`, `_deleted` are injected automatically. Defining them causes schema errors.

### 2. `AddSchema` is gone (v0.20+)
Replaced by `AddCollection`. Return is `([]CollectionVersion, error)` — discard with `_` if unneeded.

### 3. `Create` / `CreateMany` are gone (v0.20+)
Use `SaveDocument` / `SaveManyDocuments`. These are upserts — if a doc with the same docID exists, it updates; otherwise it creates.

### 4. `identity.WithContext` / `identity.FromContext` moved to `internal/identity`
The `internal` package is not importable from outside DefraDB. Use the public `acp/identity` interfaces (`FullIdentity`, `TokenIdentity`) and inject via context using node-provided helpers.

### 5. docID is content-addressed from initial data
The same initial document data always produces the same `_docID`. This is deterministic and CID-based.

### 6. GraphQL only returns requested fields
There is no wildcard `*` in DQL queries. Always list the fields you want explicitly.

### 7. Filter operators use underscore prefix
Use `_eq`, `_geq`, `_gt`, etc. — not SQL-style operators.

### 8. Mutations use `add_<TypeName>` / `update_<TypeName>` / `delete_<TypeName>`
Type names are case-sensitive and match exactly what's in the SDL.

### 9. Bearer token for HTTP goes in `Authorization` header
Format: `Authorization: bearer <jwt>`. A missing or invalid token returns `403`.

### 10. JWT `aud` must match the DefraDB host
If your node is at `localhost:9181`, the JWT `aud` field must be `"localhost:9181"`.

### 11. P2P collection ID != collection name
To subscribe to P2P updates for a collection, you need the collection's `collectionID` (a CID found in document fields), not its name.

### 12. ExecRequest returns raw `any` for GQL data
Cast carefully: `result.GQL.Data.(map[string]any)` → then access by collection name key → cast to `[]any` → iterate.

### 13. ACP with local mode doesn't work with P2P
If using Local ACP, only collections without a policy work across P2P nodes. Use SourceHub ACP for multi-node + access control.

### 14. Schema updates use PatchCollection not re-adding
You cannot re-add a collection type that already exists. Use `PatchCollection` with a JSON patch to evolve schemas.

---

## Useful CLI Reference

```bash
# Start node
defradb start

# Add collection/schema
defradb client collection add '<SDL>'
defradb client collection add -f schema.graphql

# Query
defradb client query '<GraphQL>'

# Describe collections
defradb client collection describe

# Backup
defradb client backup export backup.json
defradb client backup import backup.json

# P2P
defradb client p2p info
defradb client p2p replicator add -c <Collection> <peer-json>
defradb client p2p collection add <collectionID>

# ACP
defradb client acp policy add -f policy.yml --identity <key>
defradb client acp relationship add --collection <Name> --docID <id> --relation <rel> --actor <did> --identity <key>
defradb client acp relationship delete --collection <Name> --docID <id> --relation <rel> --actor <did> --identity <key>
```

---

## Shinzo-Specific Notes

- DefraDB is embedded (not standalone server) in both `shinzo-indexer-client` and `shinzo-host-client`
- The indexer client writes raw blockchain primitives (blocks, txs, logs) to DefraDB collections
- The host client reads those primitives via P2P pubsub, applies Lens WASM transforms, and writes to view collections
- Config is managed via `config.yaml` mounted into the container; DefraDB data in `~/data/defradb`
- The Shinzo middleware that handles bearer tokens generates them using `FullIdentity.NewToken()` and attaches via `Authorization: bearer <token>` on outbound HTTP requests to the GraphQL API
- Schema naming convention in Shinzo uses `__` (double underscore) for namespacing: e.g. `Config__LastProcessedPage`
- When querying for existing records before upsert, check `_docID` in returned results and use it in subsequent update mutations

---

*Last updated: May 2026 — covers DefraDB v0.20.x / v1.0.0-rc1*
