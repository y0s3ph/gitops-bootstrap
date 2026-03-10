package differ

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/y0s3ph/gostrap/internal/config"
	"github.com/y0s3ph/gostrap/internal/models"
)

type FileStatus int

const (
	FileIdentical FileStatus = iota
	FileModified
	FileOnlyInSource
	FileOnlyInTarget
)

type FileDiff struct {
	RelPath     string
	Status      FileStatus
	SourceLines []string
	TargetLines []string
	Hunks       []Hunk
}

type Hunk struct {
	SourceStart int
	SourceCount int
	TargetStart int
	TargetCount int
	Lines       []DiffLine
}

type DiffLine struct {
	Kind DiffKind
	Text string
}

type DiffKind int

const (
	DiffContext DiffKind = iota
	DiffAdded
	DiffRemoved
)

type DiffResult struct {
	SourceEnv string
	TargetEnv string
	AppName   string
	Files     []FileDiff
}

func (r *DiffResult) HasChanges() bool {
	for _, f := range r.Files {
		if f.Status != FileIdentical {
			return true
		}
	}
	return false
}

// Diff compares the overlay files for a given app between two environments.
// If appName is empty, it compares all apps.
func Diff(repoPath, sourceEnv, targetEnv, appName string) ([]DiffResult, error) {
	cfg, err := config.Load(repoPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if !envExists(cfg, sourceEnv) {
		return nil, fmt.Errorf("source environment %q not found in config", sourceEnv)
	}
	if !envExists(cfg, targetEnv) {
		return nil, fmt.Errorf("target environment %q not found in config", targetEnv)
	}

	apps, err := discoverApps(repoPath)
	if err != nil {
		return nil, err
	}

	if appName != "" {
		found := false
		for _, a := range apps {
			if a == appName {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("application %q not found in environments/base/", appName)
		}
		apps = []string{appName}
	}

	var results []DiffResult
	for _, app := range apps {
		result := diffApp(repoPath, sourceEnv, targetEnv, app)
		results = append(results, result)
	}

	return results, nil
}

func diffApp(repoPath, sourceEnv, targetEnv, app string) DiffResult {
	result := DiffResult{
		SourceEnv: sourceEnv,
		TargetEnv: targetEnv,
		AppName:   app,
	}

	srcDir := filepath.Join(repoPath, "environments", sourceEnv, app)
	tgtDir := filepath.Join(repoPath, "environments", targetEnv, app)

	allFiles := collectFiles(srcDir, tgtDir)

	for _, relFile := range allFiles {
		srcPath := filepath.Join(srcDir, relFile)
		tgtPath := filepath.Join(tgtDir, relFile)

		displayRel := filepath.Join("environments", "%s", app, relFile)

		fd := diffFile(srcPath, tgtPath, displayRel)
		fd.RelPath = relFile
		result.Files = append(result.Files, fd)
	}

	srcAppDef := filepath.Join(repoPath, "apps", fmt.Sprintf("%s-%s.yaml", app, sourceEnv))
	tgtAppDef := filepath.Join(repoPath, "apps", fmt.Sprintf("%s-%s.yaml", app, targetEnv))
	appDefDisplay := filepath.Join("apps", fmt.Sprintf("%s-{%s,%s}.yaml", app, sourceEnv, targetEnv))

	if fileExists(srcAppDef) || fileExists(tgtAppDef) {
		fd := diffFile(srcAppDef, tgtAppDef, appDefDisplay)
		fd.RelPath = fmt.Sprintf("apps/%s-*.yaml", app)
		result.Files = append(result.Files, fd)
	}

	return result
}

func diffFile(srcPath, tgtPath, _ string) FileDiff {
	srcExists := fileExists(srcPath)
	tgtExists := fileExists(tgtPath)

	srcLines := readLines(srcPath)
	tgtLines := readLines(tgtPath)

	if !srcExists && tgtExists {
		fd := FileDiff{Status: FileOnlyInTarget, TargetLines: tgtLines}
		fd.Hunks = buildFullHunks(tgtLines, DiffAdded)
		return fd
	}
	if srcExists && !tgtExists {
		fd := FileDiff{Status: FileOnlyInSource, SourceLines: srcLines}
		fd.Hunks = buildFullHunks(srcLines, DiffRemoved)
		return fd
	}

	if linesEqual(srcLines, tgtLines) {
		return FileDiff{Status: FileIdentical, SourceLines: srcLines, TargetLines: tgtLines}
	}

	fd := FileDiff{
		Status:      FileModified,
		SourceLines: srcLines,
		TargetLines: tgtLines,
	}
	fd.Hunks = computeHunks(srcLines, tgtLines, 3)
	return fd
}

func computeHunks(src, tgt []string, contextLines int) []Hunk {
	edits := myersDiff(src, tgt)

	type editLine struct {
		kind DiffKind
		text string
		srcN int
		tgtN int
	}

	var allLines []editLine
	si, ti := 0, 0
	for _, op := range edits {
		switch op {
		case opEqual:
			allLines = append(allLines, editLine{DiffContext, src[si], si, ti})
			si++
			ti++
		case opDelete:
			allLines = append(allLines, editLine{DiffRemoved, src[si], si, -1})
			si++
		case opInsert:
			allLines = append(allLines, editLine{DiffAdded, tgt[ti], -1, ti})
			ti++
		}
	}

	changed := make([]bool, len(allLines))
	for i, l := range allLines {
		if l.kind != DiffContext {
			changed[i] = true
		}
	}

	var groups [][2]int
	inGroup := false
	start := 0
	for i := range allLines {
		lo := max(0, i-contextLines)
		hi := min(len(allLines)-1, i+contextLines)

		if changed[i] {
			if !inGroup {
				inGroup = true
				start = lo
			}
		}
		if inGroup {
			isNearChange := false
			for j := lo; j <= hi; j++ {
				if j < len(changed) && changed[j] {
					isNearChange = true
					break
				}
			}
			if !isNearChange || i == len(allLines)-1 {
				end := min(i, len(allLines)-1)
				groups = append(groups, [2]int{start, end})
				inGroup = false
			}
		}
	}

	if inGroup {
		groups = append(groups, [2]int{start, len(allLines) - 1})
	}

	var merged [][2]int
	for _, g := range groups {
		if len(merged) > 0 && g[0] <= merged[len(merged)-1][1]+1 {
			merged[len(merged)-1][1] = g[1]
		} else {
			merged = append(merged, g)
		}
	}

	hunks := make([]Hunk, 0, len(merged))
	for _, g := range merged {
		h := Hunk{}
		for i := g[0]; i <= g[1]; i++ {
			l := allLines[i]
			h.Lines = append(h.Lines, DiffLine{Kind: l.kind, Text: l.text})
		}
		if g[0] < len(allLines) {
			first := allLines[g[0]]
			if first.srcN >= 0 {
				h.SourceStart = first.srcN + 1
			}
			if first.tgtN >= 0 {
				h.TargetStart = first.tgtN + 1
			}
		}
		srcCount, tgtCount := 0, 0
		for _, dl := range h.Lines {
			switch dl.Kind {
			case DiffContext:
				srcCount++
				tgtCount++
			case DiffRemoved:
				srcCount++
			case DiffAdded:
				tgtCount++
			}
		}
		h.SourceCount = srcCount
		h.TargetCount = tgtCount
		hunks = append(hunks, h)
	}

	return hunks
}

func buildFullHunks(lines []string, kind DiffKind) []Hunk {
	if len(lines) == 0 {
		return nil
	}
	h := Hunk{SourceStart: 1, TargetStart: 1}
	for _, l := range lines {
		h.Lines = append(h.Lines, DiffLine{Kind: kind, Text: l})
	}
	if kind == DiffAdded {
		h.TargetCount = len(lines)
	} else {
		h.SourceCount = len(lines)
	}
	return []Hunk{h}
}

func collectFiles(srcDir, tgtDir string) []string {
	seen := make(map[string]bool)

	addDir := func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() {
				seen[e.Name()] = true
			}
		}
	}

	addDir(srcDir)
	addDir(tgtDir)

	files := make([]string, 0, len(seen))
	for f := range seen {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

func envExists(cfg *models.BootstrapConfig, name string) bool {
	for _, e := range cfg.Environments {
		if e.Name == name {
			return true
		}
	}
	return false
}

func discoverApps(repoPath string) ([]string, error) {
	baseDir := filepath.Join(repoPath, "environments", "base")
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", baseDir, err)
	}

	var apps []string
	for _, e := range entries {
		if e.IsDir() {
			apps = append(apps, e.Name())
		}
	}
	return apps, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := string(data)
	if content == "" {
		return nil
	}
	return strings.Split(strings.TrimRight(content, "\n"), "\n")
}

func linesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type editOp int

const (
	opEqual  editOp = iota
	opInsert
	opDelete
)

// myersDiff computes the shortest edit script between src and tgt using
// a simplified LCS-based approach suitable for config file sizes.
func myersDiff(src, tgt []string) []editOp {
	n := len(src)
	m := len(tgt)

	if n == 0 && m == 0 {
		return nil
	}
	if n == 0 {
		ops := make([]editOp, m)
		for i := range ops {
			ops[i] = opInsert
		}
		return ops
	}
	if m == 0 {
		ops := make([]editOp, n)
		for i := range ops {
			ops[i] = opDelete
		}
		return ops
	}

	// LCS via dynamic programming — correct and simple for config-sized files
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			switch {
			case src[i-1] == tgt[j-1]:
				dp[i][j] = dp[i-1][j-1] + 1
			case dp[i-1][j] >= dp[i][j-1]:
				dp[i][j] = dp[i-1][j]
			default:
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	var ops []editOp
	i, j := n, m
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && src[i-1] == tgt[j-1]:
			ops = append(ops, opEqual)
			i--
			j--
		case i > 0 && (j == 0 || dp[i-1][j] >= dp[i][j-1]):
			ops = append(ops, opDelete)
			i--
		default:
			ops = append(ops, opInsert)
			j--
		}
	}

	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}

	return ops
}
