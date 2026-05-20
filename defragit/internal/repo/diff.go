package repo

import (
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffResult holds unified-diff output for one file.
type DiffResult struct {
	Path     string
	OldLabel string
	NewLabel string
	Hunks    string // unified diff body
	Inserted int
	Deleted  int
}

// ComputeDiff produces a unified diff between oldContent and newContent for path.
func ComputeDiff(path, oldLabel, newLabel string, oldContent, newContent []byte) DiffResult {
	old := string(oldContent)
	neu := string(newContent)

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(old, neu, true)
	dmp.DiffCleanupSemantic(diffs)

	hunks := buildUnifiedDiff(diffs, path, oldLabel, newLabel)
	ins, del := countChanges(diffs)

	return DiffResult{
		Path:     path,
		OldLabel: oldLabel,
		NewLabel: newLabel,
		Hunks:    hunks,
		Inserted: ins,
		Deleted:  del,
	}
}

// buildUnifiedDiff converts go-diff diffs into unified diff format.
func buildUnifiedDiff(diffs []diffmatchpatch.Diff, path, oldLabel, newLabel string) string {
	if len(diffs) == 0 {
		return ""
	}
	// Check if there are any changes.
	hasChanges := false
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffEqual {
			hasChanges = true
			break
		}
	}
	if !hasChanges {
		return ""
	}

	// Split diffs into lines and build hunks.
	oldLines := splitLines(reconstructOld(diffs))
	newLines := splitLines(reconstructNew(diffs))

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- a/%s\n", path)
	fmt.Fprintf(&sb, "+++ b/%s\n", path)

	// Simple line-by-line diff using the computed diffs.
	// Re-compute per-line diffs for unified output.
	dmp2 := diffmatchpatch.New()
	lineDiffs := lineModeDiffs(dmp2, strings.Join(oldLines, "\n"), strings.Join(newLines, "\n"))

	const context = 3
	writeUnifiedHunks(&sb, lineDiffs, context)

	return sb.String()
}

// lineModeDiffs computes line-level diffs.
func lineModeDiffs(dmp *diffmatchpatch.DiffMatchPatch, old, neu string) []diffmatchpatch.Diff {
	a, b, c := dmp.DiffLinesToChars(old, neu)
	diffs := dmp.DiffMain(a, b, false)
	return dmp.DiffCharsToLines(diffs, c)
}

type lineOp struct {
	op   diffmatchpatch.Operation
	text string
}

func writeUnifiedHunks(sb *strings.Builder, diffs []diffmatchpatch.Diff, ctx int) {
	// Expand diffs into lines.
	var lines []lineOp
	for _, d := range diffs {
		for _, line := range splitLines(d.Text) {
			lines = append(lines, lineOp{op: d.Type, text: line})
		}
	}

	// Find changed line ranges to group into hunks.
	changed := make([]bool, len(lines))
	for i, l := range lines {
		if l.op != diffmatchpatch.DiffEqual {
			changed[i] = true
		}
	}

	i := 0
	for i < len(lines) {
		if !changed[i] {
			i++
			continue
		}
		// Start of a hunk.
		start := i - ctx
		if start < 0 {
			start = 0
		}
		end := i
		for end < len(lines) && (changed[end] || end-i < ctx) {
			if changed[end] {
				i = end
			}
			end++
		}
		end += ctx
		if end > len(lines) {
			end = len(lines)
		}

		// Count old/new lines for the @@ header.
		oldStart, oldCount, newStart, newCount := hunkCounts(lines, start, end)
		fmt.Fprintf(sb, "@@ -%d,%d +%d,%d @@\n", oldStart+1, oldCount, newStart+1, newCount)

		for _, l := range lines[start:end] {
			switch l.op {
			case diffmatchpatch.DiffEqual:
				fmt.Fprintf(sb, " %s\n", l.text)
			case diffmatchpatch.DiffInsert:
				fmt.Fprintf(sb, "\033[32m+%s\033[0m\n", l.text)
			case diffmatchpatch.DiffDelete:
				fmt.Fprintf(sb, "\033[31m-%s\033[0m\n", l.text)
			}
		}
		i = end
	}
}

func hunkCounts(lines []lineOp, start, end int) (oldStart, oldCount, newStart, newCount int) {
	// Count preceding lines to determine line numbers.
	for i := 0; i < start; i++ {
		if lines[i].op != diffmatchpatch.DiffInsert {
			oldStart++
		}
		if lines[i].op != diffmatchpatch.DiffDelete {
			newStart++
		}
	}
	for i := start; i < end; i++ {
		if lines[i].op != diffmatchpatch.DiffInsert {
			oldCount++
		}
		if lines[i].op != diffmatchpatch.DiffDelete {
			newCount++
		}
	}
	return
}

func reconstructOld(diffs []diffmatchpatch.Diff) string {
	var sb strings.Builder
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffInsert {
			sb.WriteString(d.Text)
		}
	}
	return sb.String()
}

func reconstructNew(diffs []diffmatchpatch.Diff) string {
	var sb strings.Builder
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffDelete {
			sb.WriteString(d.Text)
		}
	}
	return sb.String()
}

func countChanges(diffs []diffmatchpatch.Diff) (inserted, deleted int) {
	for _, d := range diffs {
		lines := strings.Count(d.Text, "\n")
		if lines == 0 && d.Text != "" {
			lines = 1
		}
		switch d.Type {
		case diffmatchpatch.DiffInsert:
			inserted += lines
		case diffmatchpatch.DiffDelete:
			deleted += lines
		}
	}
	return
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	// Remove trailing empty element from a trailing newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
