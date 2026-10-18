package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type SecretFinding struct {
	Type     string `json:"type"`
	FilePath string `json:"file_path"`
	LineNum  int    `json:"line_number"`
	Masked   string `json:"masked_secret"`
}

type CodeMetrics struct {
	TotalFiles   int `json:"total_files"`
	TotalLines   int `json:"total_lines"`
	CodeLines    int `json:"code_lines"`
	CommentLines int `json:"comment_lines"`
	BlankLines   int `json:"blank_lines"`
	Functions    int `json:"functions"`
	Classes      int `json:"classes"`
	Structs      int `json:"structs"`
	Interfaces   int `json:"interfaces"`
	Constants    int `json:"constants"`
	Types        int `json:"types"`
	Imports      int `json:"imports"`
	Decorators   int `json:"decorators"`
	Async        int `json:"async_routines"`
}

type StandardFileStatus struct {
	Name     string `json:"name"`
	Found    bool   `json:"found"`
	FilePath string `json:"file_path,omitempty"`
	Priority string `json:"priority"`
}

type AuditReport struct {
	TargetDirectory string               `json:"target_directory"`
	StandardFiles   []StandardFileStatus `json:"standard_files"`
	Metrics         CodeMetrics          `json:"syntax_metrics"`
	Secrets         []SecretFinding      `json:"secret_findings"`
	Clean           bool                 `json:"is_clean"`
}

var (
	reAwsAccessKey = regexp.MustCompile(`\b(AKIA[0-9A-Z]{16})\b`)
	reAwsSecretKey = regexp.MustCompile(`(?i)(?:aws_secret_access_key|aws_secret_key|secret_access_key)\s*[:=]\s*['"]?([A-Za-z0-9/+=]{40})['"]?`)
	reGenericKey   = regexp.MustCompile(`(?i)\b(?:api_key|apikey|secret_key|auth_token|access_token|private_token|bearer)\b\s*[:=]\s*['"]?([A-Za-z0-9_\-.~+]{16,})['"]?`)
	rePassword     = regexp.MustCompile(`(?i)\b(?:password|passwd|pwd)\b\s*[:=]\s*['"]?([^\s'"]{6,})['"]?`)
	rePrivateKey   = regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`)
	reGitHubToken  = regexp.MustCompile(`\b(?:ghp|gho|ghu|ghs|ghr|github_pat)_[A-Za-z0-9_]{36,255}\b`)
	reSlackToken   = regexp.MustCompile(`\bxox[baprs]-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*\b`)
	reSlackWebhook = regexp.MustCompile(`https://hooks\.slack\.com/services/T[a-zA-Z0-9_]+/B[a-zA-Z0-9_]+/[a-zA-Z0-9_]+`)
	reGoogleApiKey = regexp.MustCompile(`\bAIza[0-9A-Za-z-_]{35}\b`)
	reStripeKey    = regexp.MustCompile(`\b(?:sk|rk)_(?:test|live)_[0-9a-zA-Z]{24,}\b`)
	reJwtToken     = regexp.MustCompile(`\beyJ[A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+\.?[A-Za-z0-9-_.+/=]*\b`)
	reOpenAiKey    = regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_-]{32,}\b`)
	reAnthropicKey = regexp.MustCompile(`\bsk-ant-api\d\d-[A-Za-z0-9_-]{32,}\b`)
	reDbConnString = regexp.MustCompile(`(?i)(?:postgres|postgresql|mysql|mongodb(?:\+srv)?|redis)://[a-zA-Z0-9_]+:[^@\s/]+@[a-zA-Z0-9_\-.:]+`)

	reFuncGo    = regexp.MustCompile(`^\s*func\s+`)
	reFuncPy    = regexp.MustCompile(`^\s*def\s+`)
	reFuncJsTs  = regexp.MustCompile(`^\s*(?:async\s+)?function\s+|(?:const|let|var)\s+\w+\s*=\s*(?:async\s*)?(?:\([^)]*\)|[a-zA-Z_]\w*)\s*=>`)
	reFuncOther = regexp.MustCompile(`^\s*(?:public|private|protected|static|\s)*(?:void|int|string|bool|auto|fn|fun)\s+\w+\s*\(`)
	reClass     = regexp.MustCompile(`^\s*(?:export\s+)?(?:default\s+)?class\s+\w+`)
	reInterface = regexp.MustCompile(`^\s*(?:type\s+\w+\s+interface|interface\s+\w+)`)
	reStruct    = regexp.MustCompile(`^\s*(?:type\s+\w+\s+struct|struct\s+\w+)`)
	reConst     = regexp.MustCompile(`^\s*(?:const\s+\w+|const\s*\(|(?:export\s+)?const\s+[A-Z_0-9]+\s*=|\benum\s+\w+)`)
	reType      = regexp.MustCompile(`^\s*(?:type\s+\w+\s+(?:string|int|float|bool|map|chan|any|func)|typedef\s+)`)
	reImport    = regexp.MustCompile(`^\s*(?:import\s+|#include\s+|using\s+|from\s+\w+\s+import|use\s+|require\()`)
	reDecorator = regexp.MustCompile(`^\s*@\w+`)
	reAsync     = regexp.MustCompile(`^\s*(?:async\s+func|async\s+def|async\s+function|\bgo\s+\w+)`)
)

var (
	auditSkipDirs    []string
	auditStrict      bool
	auditJson        bool
	auditSecretsOnly bool
	auditMetricsOnly bool
)

var defaultAuditSkips = []string{
	".git",
	"node_modules",
	"build",
	"dist",
	".next",
	".nuxt",
	".turbo",
	".cache",
	"vendor",
	"__pycache__",
	".venv",
	"venv",
	".gradle",
	"Pods",
	".idea",
	".vscode",
	".terraform",
}

var skipFileNames = map[string]bool{
	"go.sum":            true,
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	"Cargo.lock":        true,
	"composer.lock":     true,
	"poetry.lock":       true,
	"flake.lock":        true,
}

var auditCmd = &cobra.Command{
	Use:   "audit [path]",
	Short: "Scan codebase for exposed secrets, standard repository files, and syntax metrics",
	Run:   runAudit,
}

func init() {
	auditCmd.Flags().StringSliceVarP(&auditSkipDirs, "skip", "s", []string{}, "Folders or files to skip")
	auditCmd.Flags().BoolVar(&auditStrict, "strict", false, "Exit with non-zero exit code if secrets are discovered")
	auditCmd.Flags().BoolVar(&auditJson, "json", false, "Output results in JSON format")
	auditCmd.Flags().BoolVar(&auditSecretsOnly, "secrets-only", false, "Only run and display secrets audit")
	auditCmd.Flags().BoolVar(&auditMetricsOnly, "metrics-only", false, "Only run and display codebase syntax metrics")
	RootCmd.AddCommand(auditCmd)
}

func runAudit(cmd *cobra.Command, args []string) {
	var targetDir string
	if len(args) == 0 {
		pwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current working directory: %v", err)
			return
		}
		targetDir = pwd
	} else {
		targetDir = args[0]
	}

	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		color.Red("Invalid path: %v", err)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		color.Red("Path does not exist: %v", err)
		return
	}

	if !info.IsDir() {
		color.Red("Target path must be a directory")
		return
	}

	var findings []SecretFinding
	var metrics CodeMetrics

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 64)
	var filesScanned int64

	stopSpinner := make(chan struct{})
	if !auditJson {
		fmt.Println()
		fmt.Printf("🔍 Auditing: %s\n\n", color.YellowString(absPath))

		go func() {
			spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			i := 0
			for {
				select {
				case <-stopSpinner:
					fmt.Print("\r\033[2K\r")
					return
				case <-time.After(80 * time.Millisecond):
					count := atomic.LoadInt64(&filesScanned)
					fmt.Printf("\r\033[2K  %s Auditing codebase... (%d files scanned)", color.CyanString(spinChars[i%len(spinChars)]), count)
					i++
				}
			}
		}()
	}

	sem <- struct{}{}
	wg.Add(1)
	go func() {
		defer func() { <-sem }()
		defer wg.Done()
		scanAuditDir(absPath, &findings, &metrics, &mu, &wg, sem, auditSkipDirs, &filesScanned)
	}()
	wg.Wait()

	if !auditJson {
		close(stopSpinner)
		time.Sleep(30 * time.Millisecond)
		fmt.Print("\r\033[2K\r")
	}

	standardFiles := checkStandardRepositoryFiles(absPath)
	isClean := len(findings) == 0

	if auditJson {
		report := AuditReport{
			TargetDirectory: absPath,
			StandardFiles:   standardFiles,
			Metrics:         metrics,
			Secrets:         findings,
			Clean:           isClean,
		}
		jsonBytes, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			color.Red("Error marshaling json: %v", err)
			return
		}
		fmt.Println(string(jsonBytes))
		if auditStrict && !isClean {
			os.Exit(1)
		}
		return
	}

	if !auditMetricsOnly {
		renderStandardFilesBox(standardFiles)
	}

	if !auditSecretsOnly {
		renderMetricsBox(metrics)
	}

	if !auditMetricsOnly {
		renderSecretsBox(findings)
	}

	if auditStrict && !isClean {
		os.Exit(1)
	}
}

func checkStandardRepositoryFiles(root string) []StandardFileStatus {
	checks := []struct {
		name     string
		priority string
		patterns []string
	}{
		{"README", "High", []string{"README.md", "README", "README.txt", "readme.md"}},
		{"LICENSE", "High", []string{"LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING"}},
		{".gitignore", "High", []string{".gitignore"}},
		{"CONTRIBUTING", "Medium", []string{"CONTRIBUTING.md", "CONTRIBUTING", "contributing.md"}},
		{"CODE_OF_CONDUCT", "Medium", []string{"CODE_OF_CONDUCT.md", "CODE_OF_CONDUCT", "code_of_conduct.md"}},
		{"SECURITY", "High", []string{"SECURITY.md", "SECURITY", "security.md"}},
		{"CHANGELOG", "Low", []string{"CHANGELOG.md", "CHANGELOG", "changelog.md"}},
	}

	var results []StandardFileStatus
	for _, c := range checks {
		found := false
		matchedPath := ""
		for _, pat := range c.patterns {
			p := filepath.Join(root, pat)
			if _, err := os.Stat(p); err == nil {
				found = true
				matchedPath = pat
				break
			}
		}
		results = append(results, StandardFileStatus{
			Name:     c.name,
			Found:    found,
			FilePath: matchedPath,
			Priority: c.priority,
		})
	}
	return results
}

func renderStandardFilesBox(files []StandardFileStatus) {
	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│                  STANDARD REPOSITORY FILES                  │")
	color.Cyan("├───────────────────────┬──────────┬──────────────────────────┤")
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-24s "+color.CyanString("│\n"), "Standard File", "Status", "Details")
	color.Cyan("├───────────────────────┼──────────┼──────────────────────────┤")

	for _, f := range files {
		statusStr := color.GreenString("  ✅    ")
		detailStr := color.HiBlackString(f.FilePath)
		if !f.Found {
			statusStr = color.RedString("  ❌    ")
			if f.Priority == "High" {
				detailStr = color.HiRedString("[Missing - Priority]")
			} else {
				detailStr = color.YellowString("[Missing]")
			}
		}
		fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+"%s"+color.CyanString("│")+" %-33s "+color.CyanString("│\n"), f.Name, statusStr, detailStr)
	}
	color.Cyan("╰───────────────────────┴──────────┴──────────────────────────╯\n")
}

func renderMetricsBox(m CodeMetrics) {
	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│              CODEBASE STRUCTURE & SYNTAX METRICS            │")
	color.Cyan("├──────────────────────────────┬──────────────────────────────┤")
	fmt.Printf(color.CyanString("│")+" %-28s "+color.CyanString("│")+" %-28s "+color.CyanString("│\n"), "Metric Category", "Scanned Quantity")
	color.Cyan("├──────────────────────────────┼──────────────────────────────┤")

	fmt.Printf(color.CyanString("│")+" %-28s "+color.CyanString("│")+" %-28s "+color.CyanString("│\n"), "Total Source Files", color.HiYellowString("%d", m.TotalFiles))
	fmt.Printf(color.CyanString("│")+" %-28s "+color.CyanString("│")+" %-28s "+color.CyanString("│\n"), "Total Lines of Code (LOC)", color.HiWhiteString("%d", m.TotalLines))

	var codePct, commentPct, blankPct float64
	if m.TotalLines > 0 {
		codePct = (float64(m.CodeLines) / float64(m.TotalLines)) * 100
		commentPct = (float64(m.CommentLines) / float64(m.TotalLines)) * 100
		blankPct = (float64(m.BlankLines) / float64(m.TotalLines)) * 100
	}

	fmt.Printf(color.CyanString("│")+"   ├── Executable Code Lines   "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.GreenString("%d (%.1f%%)", m.CodeLines, codePct))
	fmt.Printf(color.CyanString("│")+"   ├── Comment Lines           "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiBlackString("%d (%.1f%%)", m.CommentLines, commentPct))
	fmt.Printf(color.CyanString("│")+"   └── Blank Lines             "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiBlackString("%d (%.1f%%)", m.BlankLines, blankPct))

	color.Cyan("├──────────────────────────────┼──────────────────────────────┤")
	fmt.Printf(color.CyanString("│")+" Functions & Methods           "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiGreenString("%d", m.Functions))
	fmt.Printf(color.CyanString("│")+" Classes & Components          "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiCyanString("%d", m.Classes))
	fmt.Printf(color.CyanString("│")+" Structs & Records             "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiYellowString("%d", m.Structs))
	fmt.Printf(color.CyanString("│")+" Interfaces & Protocols        "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiMagentaString("%d", m.Interfaces))
	fmt.Printf(color.CyanString("│")+" Constants & Enums             "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiGreenString("%d", m.Constants))
	fmt.Printf(color.CyanString("│")+" Types & Type Aliases          "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiCyanString("%d", m.Types))
	fmt.Printf(color.CyanString("│")+" Imports & External Modules    "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiYellowString("%d", m.Imports))
	fmt.Printf(color.CyanString("│")+" Decorators & Annotations      "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiMagentaString("%d", m.Decorators))
	fmt.Printf(color.CyanString("│")+" Async / Coroutine Routines    "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiGreenString("%d", m.Async))

	color.Cyan("├──────────────────────────────┼──────────────────────────────┤")
	avgLoc := 0.0
	if m.Functions > 0 {
		avgLoc = float64(m.CodeLines) / float64(m.Functions)
	}
	ratio := 0.0
	if m.CommentLines > 0 {
		ratio = float64(m.CodeLines) / float64(m.CommentLines)
	}

	fmt.Printf(color.CyanString("│")+" Avg Lines Per Function        "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiWhiteString("%.1f lines", avgLoc))
	fmt.Printf(color.CyanString("│")+" Code-to-Comment Ratio         "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiWhiteString("%.1f : 1", ratio))

	color.Cyan("╰──────────────────────────────┴──────────────────────────────╯\n")
}

func renderSecretsBox(findings []SecretFinding) {
	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│                 SECURITY AUDIT & SECRET LEAKS               │")
	color.Cyan("├─────────────────────────────────────────────────────────────┤")

	if len(findings) == 0 {
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.GreenString("🔒 Clean Repository! No exposed credentials or secrets found."))
		color.Cyan("╰─────────────────────────────────────────────────────────────╯\n")
		return
	}

	alertMsg := fmt.Sprintf("🚨 %d Security Vulnerability / Secret Leak(s) Detected:", len(findings))
	fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.RedString(alertMsg))
	fmt.Printf(color.CyanString("│")+" %-60s "+color.CyanString("│\n"), "")

	for i, f := range findings {
		typeStr := color.HiRedString("[%s]", f.Type)
		locStr := fmt.Sprintf("%d. %s at %s:%d", i+1, typeStr, f.FilePath, f.LineNum)
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), locStr)
		valStr := fmt.Sprintf("   └── Masked: %s", color.HiMagentaString(f.Masked))
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), valStr)
		if i < len(findings)-1 {
			fmt.Printf(color.CyanString("│")+" %-60s "+color.CyanString("│\n"), "")
		}
	}

	color.Cyan("╰─────────────────────────────────────────────────────────────╯\n")
}

func scanAuditDir(currentDir string, findings *[]SecretFinding, metrics *CodeMetrics, mu *sync.Mutex, wg *sync.WaitGroup, sem chan struct{}, skips []string, scannedCount *int64) {
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if isAuditSkipped(name, skips) {
			continue
		}

		fullPath := filepath.Join(currentDir, name)

		if entry.IsDir() {
			select {
			case sem <- struct{}{}:
				wg.Add(1)
				go func(p string) {
					defer func() { <-sem }()
					defer wg.Done()
					scanAuditDir(p, findings, metrics, mu, wg, sem, skips, scannedCount)
				}(fullPath)
			default:
				scanAuditDir(fullPath, findings, metrics, mu, wg, sem, skips, scannedCount)
			}
		} else {
			if isCandidateTextFile(name, fullPath) {
				atomic.AddInt64(scannedCount, 1)
				select {
				case sem <- struct{}{}:
					wg.Add(1)
					go func(p string) {
						defer func() { <-sem }()
						defer wg.Done()
						auditFile(p, findings, metrics, mu)
					}(fullPath)
				default:
					auditFile(fullPath, findings, metrics, mu)
				}
			}
		}
	}
}

func isAuditSkipped(name string, skips []string) bool {
	if skipFileNames[name] {
		return true
	}
	for _, s := range defaultAuditSkips {
		if s == name {
			return true
		}
	}
	for _, s := range skips {
		if s == name {
			return true
		}
	}
	return false
}

func isCandidateTextFile(name, path string) bool {
	if skipFileNames[name] {
		return false
	}

	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".min.js") || strings.HasSuffix(lower, ".min.css") || strings.HasSuffix(lower, ".map") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(name))
	binaryExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true,
		".svg": true, ".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".exe": true, ".bin": true, ".iso": true, ".dll": true, ".so": true,
		".dylib": true, ".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
		".mp4": true, ".mp3": true, ".wav": true, ".db": true, ".sqlite": true,
		".wasm": true,
	}

	if binaryExts[ext] {
		return false
	}

	info, err := os.Stat(path)
	if err != nil || info.Size() > 2*1024*1024 {
		return false
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}

	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return false
		}
	}

	contentType := http.DetectContentType(buf[:n])
	if strings.HasPrefix(contentType, "text/") || strings.Contains(contentType, "json") || strings.Contains(contentType, "xml") || strings.Contains(contentType, "javascript") {
		return true
	}

	return true
}

func auditFile(path string, findings *[]SecretFinding, metrics *CodeMetrics, mu *sync.Mutex) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	var fileFindings []SecretFinding
	var total, code, comment, blank int
	var funcs, classes, structs, interfaces, consts, types, imports, decorators, async int

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 512*1024)

	lineNum := 1
	inBlockComment := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		total++

		if trimmed == "" {
			blank++
		} else if inBlockComment {
			comment++
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
		} else if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "--") {
			comment++
		} else if strings.HasPrefix(trimmed, "/*") {
			comment++
			if !strings.Contains(trimmed, "*/") {
				inBlockComment = true
			}
		} else {
			code++
		}

		if reFuncGo.MatchString(line) || reFuncPy.MatchString(line) || reFuncJsTs.MatchString(line) || reFuncOther.MatchString(line) {
			funcs++
		}
		if reClass.MatchString(line) {
			classes++
		}
		if reStruct.MatchString(line) {
			structs++
		}
		if reInterface.MatchString(line) {
			interfaces++
		}
		if reConst.MatchString(line) {
			consts++
		}
		if reType.MatchString(line) {
			types++
		}
		if reImport.MatchString(line) {
			imports++
		}
		if reDecorator.MatchString(line) {
			decorators++
		}
		if reAsync.MatchString(line) {
			async++
		}

		checkSecretMatches(line, path, lineNum, &fileFindings)
		lineNum++
	}

	mu.Lock()
	metrics.TotalFiles++
	metrics.TotalLines += total
	metrics.CodeLines += code
	metrics.CommentLines += comment
	metrics.BlankLines += blank
	metrics.Functions += funcs
	metrics.Classes += classes
	metrics.Structs += structs
	metrics.Interfaces += interfaces
	metrics.Constants += consts
	metrics.Types += types
	metrics.Imports += imports
	metrics.Decorators += decorators
	metrics.Async += async
	if len(fileFindings) > 0 {
		*findings = append(*findings, fileFindings...)
	}
	mu.Unlock()
}

func checkSecretMatches(line, path string, lineNum int, findings *[]SecretFinding) {
	if m := reAwsAccessKey.FindStringSubmatch(line); len(m) > 1 {
		*findings = append(*findings, SecretFinding{
			Type:     "AWS Access Key",
			FilePath: path,
			LineNum:  lineNum,
			Masked:   maskSecret(m[1]),
		})
	}
	if m := reAwsSecretKey.FindStringSubmatch(line); len(m) > 1 {
		*findings = append(*findings, SecretFinding{
			Type:     "AWS Secret Key",
			FilePath: path,
			LineNum:  lineNum,
			Masked:   maskSecret(m[1]),
		})
	}
	if m := reGitHubToken.FindString(line); m != "" {
		*findings = append(*findings, SecretFinding{
			Type:     "GitHub Token",
			FilePath: path,
			LineNum:  lineNum,
			Masked:   maskSecret(m),
		})
	}
	if m := reSlackWebhook.FindString(line); m != "" {
		*findings = append(*findings, SecretFinding{
			Type:     "Slack Webhook",
			FilePath: path,
			LineNum:  lineNum,
			Masked:   maskSecret(m),
		})
	}
	if m := reSlackToken.FindString(line); m != "" {
		*findings = append(*findings, SecretFinding{
			Type:     "Slack Token",
			FilePath: path,
			LineNum:  lineNum,
			Masked:   maskSecret(m),
		})
	}
	if m := reGoogleApiKey.FindString(line); m != "" {
		*findings = append(*findings, SecretFinding{
			Type:     "Google API Key",
			FilePath: path,
			LineNum:  lineNum,
			Masked:   maskSecret(m),
		})
	}
	if m := reStripeKey.FindString(line); m != "" {
		*findings = append(*findings, SecretFinding{
			Type:     "Stripe API Key",
			FilePath: path,
			LineNum:  lineNum,
			Masked:   maskSecret(m),
		})
	}
	if m := reOpenAiKey.FindString(line); m != "" {
		*findings = append(*findings, SecretFinding{
			Type:     "OpenAI Key",
			FilePath: path,
			LineNum:  lineNum,
			Masked:   maskSecret(m),
		})
	}
	if m := reAnthropicKey.FindString(line); m != "" {
		*findings = append(*findings, SecretFinding{
			Type:     "Anthropic Key",
			FilePath: path,
			LineNum:  lineNum,
			Masked:   maskSecret(m),
		})
	}
	if m := reDbConnString.FindString(line); m != "" {
		*findings = append(*findings, SecretFinding{
			Type:     "Database Connection String",
			FilePath: path,
			LineNum:  lineNum,
			Masked:   maskSecret(m),
		})
	}
	if rePrivateKey.MatchString(line) {
		*findings = append(*findings, SecretFinding{
			Type:     "Private Key Header",
			FilePath: path,
			LineNum:  lineNum,
			Masked:   "-----BEGIN PRIVATE KEY... [REDACTED]",
		})
	}
	if m := reGenericKey.FindStringSubmatch(line); len(m) > 1 {
		val := m[1]
		if !isFalsePositiveGeneric(val) {
			*findings = append(*findings, SecretFinding{
				Type:     "Generic API Secret",
				FilePath: path,
				LineNum:  lineNum,
				Masked:   maskSecret(val),
			})
		}
	}
	if m := rePassword.FindStringSubmatch(line); len(m) > 1 {
		val := m[1]
		if !isFalsePositivePassword(val) {
			*findings = append(*findings, SecretFinding{
				Type:     "Hardcoded Password",
				FilePath: path,
				LineNum:  lineNum,
				Masked:   maskSecret(val),
			})
		}
	}
}

func isFalsePositiveGeneric(val string) bool {
	lower := strings.ToLower(val)
	if strings.HasPrefix(lower, "regexp.") || strings.HasPrefix(lower, "os.getenv") || strings.HasPrefix(lower, "env(") || strings.HasSuffix(lower, "()") {
		return true
	}
	commonPlaceholders := []string{
		"your_api_key", "your-api-key", "xxxx", "test", "demo", "placeholder", "changeme", "example",
	}
	for _, p := range commonPlaceholders {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func isFalsePositivePassword(val string) bool {
	lower := strings.ToLower(val)
	if strings.HasPrefix(lower, "regexp.") || strings.HasPrefix(lower, "os.getenv") || strings.HasPrefix(lower, "env(") || strings.HasSuffix(lower, "()") {
		return true
	}
	placeholders := []string{
		"password", "secret", "xxxx", "123456", "admin", "root", "example", "changeme", "dummy",
	}
	for _, p := range placeholders {
		if lower == p {
			return true
		}
	}
	return false
}

func maskSecret(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 6 {
		return "******"
	}
	prefix := s[:4]
	suffix := ""
	if len(s) > 12 {
		suffix = s[len(s)-2:]
	}
	maskedLen := len(s) - len(prefix) - len(suffix)
	if maskedLen < 4 {
		maskedLen = 4
	}
	return prefix + strings.Repeat("*", maskedLen) + suffix
}
