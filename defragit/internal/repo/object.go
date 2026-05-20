package repo

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/duncanbrown/defragit/internal/db"
	"github.com/sourcenetwork/defradb/node"
)

// BlobInfo holds metadata about a stored blob.
type BlobInfo struct {
	DocID       string
	ContentHash string
	Encoding    string
}

// StoreFile reads a file from disk and stores it as a deduplicated blob.
func StoreFile(ctx context.Context, n *node.Node, filePath string) (*BlobInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}
	return StoreBytes(ctx, n, data)
}

// StoreBytes stores raw bytes as a deduplicated blob.
func StoreBytes(ctx context.Context, n *node.Node, data []byte) (*BlobInfo, error) {
	sum := sha256.Sum256(data)
	hash := fmt.Sprintf("%x", sum)

	// Check for existing blob with the same content hash.
	existing, err := FindBlob(ctx, n, hash)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	encoding := "utf8"
	content := string(data)
	if !utf8.Valid(data) {
		encoding = "base64"
		content = base64.StdEncoding.EncodeToString(data)
	}

	mutation := fmt.Sprintf(`mutation {
		add_DG__Blob(input: {
			contentHash: %q,
			content: %q,
			size: %d,
			encoding: %q
		}) {
			_docID
		}
	}`, hash, content, len(data), encoding)

	result := n.DB.ExecRequest(ctx, mutation)
	docs, err := db.ExtractDocs(result, "add_DG__Blob")
	if err != nil {
		return nil, fmt.Errorf("storing blob: %w", err)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no docID returned from blob creation")
	}
	return &BlobInfo{
		DocID:       db.Str(docs[0], "_docID"),
		ContentHash: hash,
		Encoding:    encoding,
	}, nil
}

// FindBlob returns an existing blob by content hash, or nil if not found.
func FindBlob(ctx context.Context, n *node.Node, contentHash string) (*BlobInfo, error) {
	query := fmt.Sprintf(`query {
		DG__Blob(filter: {contentHash: {_eq: %q}}) {
			_docID
			contentHash
			encoding
		}
	}`, contentHash)
	result := n.DB.ExecRequest(ctx, query)
	docs, err := db.ExtractDocs(result, "DG__Blob")
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	return &BlobInfo{
		DocID:       db.Str(docs[0], "_docID"),
		ContentHash: db.Str(docs[0], "contentHash"),
		Encoding:    db.Str(docs[0], "encoding"),
	}, nil
}

// ReadBlob retrieves and decodes blob content by content hash.
func ReadBlob(ctx context.Context, n *node.Node, contentHash string) ([]byte, error) {
	query := fmt.Sprintf(`query {
		DG__Blob(filter: {contentHash: {_eq: %q}}) {
			content
			encoding
		}
	}`, contentHash)
	result := n.DB.ExecRequest(ctx, query)
	docs, err := db.ExtractDocs(result, "DG__Blob")
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("blob not found: %s", contentHash)
	}
	content := db.Str(docs[0], "content")
	if db.Str(docs[0], "encoding") == "base64" {
		return base64.StdEncoding.DecodeString(content)
	}
	return []byte(content), nil
}

// ContentHash computes the sha256 hex hash of data.
func ContentHash(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}