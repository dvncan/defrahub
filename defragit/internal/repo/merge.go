package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sourcenetwork/defradb/node"
)

// MergeResult describes the outcome of a three-way merge.
type MergeResult struct {
	// Files with clean merges.
	Merged map[string][]byte
	// Files with conflicts — value is the conflict-marked content.
	Conflicts map[string][]byte
}

// HasConflicts returns true if any file had a merge conflict.
func (m *MergeResult) HasConflicts() bool {
	return len(m.Conflicts) > 0
}

// ThreeWayMerge performs a three-way merge between ours and theirs using base as the common ancestor.
// All three are map[filePath]contentHash.
func ThreeWayMerge(
	ctx context.Context,
	n *node.Node,
	base, ours, theirs map[string]string,
) (*MergeResult, error) {
	result := &MergeResult{
		Merged:    make(map[string][]byte),
		Conflicts: make(map[string][]byte),
	}

	// Union of all file paths.
	paths := unionKeys(base, ours, theirs)

	for _, path := range paths {
		baseHash := base[path]
		ourHash := ours[path]
		theirHash := theirs[path]

		// No change on either side.
		if ourHash == theirHash {
			if ourHash != "" {
				content, err := ReadBlob(ctx, n, ourHash)
				if err != nil {
					return nil, err
				}
				result.Merged[path] = content
			}
			continue
		}

		ourChanged := ourHash != baseHash
		theirChanged := theirHash != baseHash

		switch {
		case !ourChanged && theirChanged:
			// Only they changed — take theirs.
			if theirHash != "" {
				content, err := ReadBlob(ctx, n, theirHash)
				if err != nil {
					return nil, err
				}
				result.Merged[path] = content
			}
		case ourChanged && !theirChanged:
			// Only we changed — take ours.
			if ourHash != "" {
				content, err := ReadBlob(ctx, n, ourHash)
				if err != nil {
					return nil, err
				}
				result.Merged[path] = content
			}
		default:
			// Both changed — attempt a three-way text merge.
			baseContent, _ := readOrEmpty(ctx, n, baseHash)
			ourContent, _ := readOrEmpty(ctx, n, ourHash)
			theirContent, _ := readOrEmpty(ctx, n, theirHash)

			merged, conflict := mergeText(baseContent, ourContent, theirContent, path)
			if conflict {
				result.Conflicts[path] = merged
			} else {
				result.Merged[path] = merged
			}
		}
	}

	return result, nil
}

// FindCommonAncestor finds the first shared commit docID between two branches
// by walking both commit chains simultaneously.
func FindCommonAncestor(ctx context.Context, n *node.Node, headA, headB string) (string, error) {
	// Build the full ancestor set of headA.
	ancestorsA := map[string]bool{}
	cur := headA
	for cur != "" {
		ancestorsA[cur] = true
		c, err := GetCommit(ctx, n, cur)
		if err != nil {
			return "", err
		}
		if c == nil {
			break
		}
		cur = c.ParentCID
	}

	// Walk headB until we hit a commit that's in ancestorsA.
	cur = headB
	for cur != "" {
		if ancestorsA[cur] {
			return cur, nil
		}
		c, err := GetCommit(ctx, n, cur)
		if err != nil {
			return "", err
		}
		if c == nil {
			break
		}
		cur = c.ParentCID
	}
	return "", nil // no common ancestor
}

// mergeText does a simple three-way text merge.
// Returns the merged content and true if there were conflicts.
func mergeText(base, ours, theirs []byte, path string) ([]byte, bool) {
	baseStr := string(base)
	ourStr := string(ours)
	theirStr := string(theirs)

	dmp := diffmatchpatch.New()

	// Apply ours patch.
	oursPatches := dmp.PatchMake(baseStr, ourStr)
	// Apply theirs patch.
	theirsPatches := dmp.PatchMake(baseStr, theirStr)

	// Try applying both patches to base.
	result1, applied1 := dmp.PatchApply(oursPatches, baseStr)
	result2, applied2 := dmp.PatchApply(theirsPatches, result1)

	oursOK := allApplied(applied1)
	theirsOK := allApplied(applied2)

	if oursOK && theirsOK {
		return []byte(result2), false
	}

	// Conflict — produce conflict markers.
	var sb strings.Builder
	fmt.Fprintf(&sb, "<<<<<<< HEAD\n%s\n=======\n%s\n>>>>>>> theirs\n", ourStr, theirStr)
	return []byte(sb.String()), true
}

func allApplied(applied []bool) bool {
	for _, ok := range applied {
		if !ok {
			return false
		}
	}
	return true
}

func readOrEmpty(ctx context.Context, n *node.Node, hash string) ([]byte, error) {
	if hash == "" {
		return []byte{}, nil
	}
	return ReadBlob(ctx, n, hash)
}

func unionKeys(maps ...map[string]string) []string {
	seen := map[string]bool{}
	var keys []string
	for _, m := range maps {
		for k := range m {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	return keys
}
