package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/V3DxNT/Ray/internal/fsutil"
	"github.com/V3DxNT/Ray/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type EnvUsageOccurrence struct {
	FilePath string `json:"file_path"`
	LineNum  int    `json:"line_number"`
}

type MissingEnvReportItem struct {
	Variable    string               `json:"variable"`
	Occurrences []EnvUsageOccurrence `json:"occurrences"`
}

type EnvAuditReport struct {
	TargetDirectory string                 `json:"target_directory"`
	EnvFile         string                 `json:"env_file"`
	EnvFileFound    bool                   `json:"env_file_found"`
	DefinedCount    int                    `json:"defined_count"`
	DetectedCount   int                    `json:"detected_count"`
	MissingCount    int                    `json:"missing_count"`
	Missing         []MissingEnvReportItem `json:"missing"`
	Synced          bool                   `json:"is_synced"`
}

var (
	reNodeEnv1 = regexp.MustCompile(`\bprocess\.env\.([A-Z0-9_]+)\b`)
	reNodeEnv2 = regexp.MustCompile(`\bprocess\.env\[['"]([A-Z0-9_]+)['"]\]`)
	reGoEnv1   = regexp.MustCompile(`\bos\.Getenv\(['"]([A-Z0-9_]+)['"]\)`)
	reGoEnv2   = regexp.MustCompile(`\bos\.LookupEnv\(['"]([A-Z0-9_]+)['"]\)`)
	rePyEnv1   = regexp.MustCompile(`\bos\.environ\.get\(['"]([A-Z0-9_]+)['"]\)`)
	rePyEnv2   = regexp.MustCompile(`\bos\.environ\[['"]([A-Z0-9_]+)['"]\]`)
	rePyEnv3   = regexp.MustCompile(`\bos\.getenv\(['"]([A-Z0-9_]+)['"]\)`)
	reRustEnv  = regexp.MustCompile(`\b(?:std::)?env::var\(['"]([A-Z0-9_]+)['"]\)`)
	reJavaEnv  = regexp.MustCompile(`\bSystem\.getenv\(['"]([A-Z0-9_]+)['"]\)`)
	rePhpEnv   = regexp.MustCompile(`\b(?:getenv|\$_ENV)\[?['"]([A-Z0-9_]+)['"]\]?`)
	reRubyEnv  = regexp.MustCompile(`\bENV\[['"]([A-Z0-9_]+)['"]\]`)
)

var (
	envFileCustom string
	envStrict     bool
	envJson       bool
	envFix        bool
	envSkipDirs   []string
)

var envCmd = &cobra.Command{
	Use:   "env [path]",
	Short: "Find environment variables used in code and check against .env",
	Long:  "Scan your code across JavaScript, TypeScript, Go, Python, Rust, Java, and PHP to find environment variable lookups (like process.env.API_KEY or os.Getenv(\"DB_HOST\")). Then verify that every variable is defined in your .env file so nothing breaks at runtime.",
	Example: `  ray env
  ray env --file .env.local
  ray env --fix
  ray env --strict
  ray env --json
  ray env --skip tests,fixtures`,
	Run: runEnvAudit,
}

func init() {
	RootCmd.AddCommand(envCmd)
	envCmd.Flags().StringVarP(&envFileCustom, "file", "f", ".env", "Path to the environment file to check against")
	envCmd.Flags().BoolVarP(&envStrict, "strict", "s", false, "Exit with an error code if any missing environment variables are found")
	envCmd.Flags().BoolVar(&envJson, "json", false, "Output results in JSON format")
	envCmd.Flags().BoolVar(&envFix, "fix", false, "Automatically add missing variables with blank placeholders to .env")
	envCmd.Flags().StringSliceVar(&envSkipDirs, "skip", []string{}, "Additional folders or files to skip")
}

func runEnvAudit(cmd *cobra.Command, args []string) {
	target := "."
	if len(args) > 0 {
		target = args[0]
	}

	absPath, err := filepath.Abs(target)
	if err != nil {
		color.Red("Failed to resolve target directory: %v", err)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		color.Red("Target path must be an existing directory")
		return
	}

	report := PerformEnvAudit(absPath, envFileCustom, envSkipDirs, envJson)

	if envJson {
		jsonBytes, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			color.Red("Error marshaling json: %v", err)
			return
		}
		fmt.Println(string(jsonBytes))
		if envStrict && !report.Synced {
			os.Exit(1)
		}
		return
	}

	renderEnvAuditBox(report)

	if len(report.Missing) > 0 {
		fmt.Println()
		for _, item := range report.Missing {
			for _, occ := range item.Occurrences {
				relFile := occ.FilePath
				if rel, err := filepath.Rel(absPath, occ.FilePath); err == nil && !strings.HasPrefix(rel, "..") {
					relFile = rel
				}
				fmt.Printf("%s Missing from %s: %s (Found in %s:%d)\n",
					color.RedString("❌"),
					color.HiYellowString(filepath.Base(report.EnvFile)),
					color.HiWhiteString(item.Variable),
					color.CyanString(relFile),
					occ.LineNum,
				)
			}
		}
		fmt.Println()

		if envFix && report.EnvFileFound {
			fixMissingEnvVars(report.EnvFile, report.Missing)
		}
	} else {
		fmt.Println()
		color.HiGreen("✔ All environment variables used in codebase are defined in %s!", filepath.Base(report.EnvFile))
		fmt.Println()
	}

	if envStrict && !report.Synced {
		os.Exit(1)
	}
}

func PerformEnvAudit(root string, envFileName string, customSkips []string, silentSpinner bool) EnvAuditReport {
	resolvedEnvPath, foundEnvKeys, envExists := parseLocalEnv(root, envFileName)

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 64)

	detectedVars := make(map[string][]EnvUsageOccurrence)

	spinner := ui.NewSpinner("Scanning codebase...")
	if !silentSpinner {
		fmt.Println()
		fmt.Printf("🔍 Scanning for environment variables: %s\n\n", color.YellowString(root))
		spinner.Start()
	}

	sem <- struct{}{}
	wg.Add(1)
	go func() {
		defer func() { <-sem }()
		defer wg.Done()
		scanEnvDir(root, detectedVars, &mu, &wg, sem, customSkips, spinner)
	}()
	wg.Wait()

	if !silentSpinner {
		spinner.Stop()
	}

	var missingList []MissingEnvReportItem
	for varName, occurrences := range detectedVars {
		if !foundEnvKeys[varName] {
			missingList = append(missingList, MissingEnvReportItem{
				Variable:    varName,
				Occurrences: occurrences,
			})
		}
	}

	return EnvAuditReport{
		TargetDirectory: root,
		EnvFile:         resolvedEnvPath,
		EnvFileFound:    envExists,
		DefinedCount:    len(foundEnvKeys),
		DetectedCount:   len(detectedVars),
		MissingCount:    len(missingList),
		Missing:         missingList,
		Synced:          len(missingList) == 0,
	}
}

func parseLocalEnv(root, envFileName string) (string, map[string]bool, bool) {
	foundEnv := make(map[string]bool)

	targetFile := filepath.Join(root, envFileName)
	if !fsutil.FileExists(targetFile) {
		fallbacks := []string{".env", ".env.local", ".env.development", ".env.example"}
		foundFallback := false
		for _, fb := range fallbacks {
			candidate := filepath.Join(root, fb)
			if fsutil.FileExists(candidate) {
				targetFile = candidate
				foundFallback = true
				break
			}
		}
		if !foundFallback {
			return targetFile, foundEnv, false
		}
	}

	file, err := os.Open(targetFile)
	if err != nil {
		return targetFile, foundEnv, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) > 0 {
			key := strings.TrimSpace(parts[0])
			if key != "" {
				foundEnv[key] = true
			}
		}
	}

	return targetFile, foundEnv, true
}

func scanEnvDir(currentDir string, detectedVars map[string][]EnvUsageOccurrence, mu *sync.Mutex, wg *sync.WaitGroup, sem chan struct{}, skips []string, spinner *ui.Spinner) {
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".env") || fsutil.ShouldSkip(name, skips) {
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
					scanEnvDir(p, detectedVars, mu, wg, sem, skips, spinner)
				}(fullPath)
			default:
				scanEnvDir(fullPath, detectedVars, mu, wg, sem, skips, spinner)
			}
		} else {
			if fsutil.IsCandidateCodeFile(name, fullPath) {
				spinner.Increment()
				select {
				case sem <- struct{}{}:
					wg.Add(1)
					go func(p string) {
						defer func() { <-sem }()
						defer wg.Done()
						scanEnvFile(p, detectedVars, mu)
					}(fullPath)
				default:
					scanEnvFile(fullPath, detectedVars, mu)
				}
			}
		}
	}
}

func scanEnvFile(filePath string, detectedVars map[string][]EnvUsageOccurrence, mu *sync.Mutex) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 1

	for scanner.Scan() {
		line := scanner.Text()

		var matchedKeys []string
		switch ext {
		case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
			matchedKeys = append(matchedKeys, extractRegexMatches(reNodeEnv1, line)...)
			matchedKeys = append(matchedKeys, extractRegexMatches(reNodeEnv2, line)...)
		case ".go":
			matchedKeys = append(matchedKeys, extractRegexMatches(reGoEnv1, line)...)
			matchedKeys = append(matchedKeys, extractRegexMatches(reGoEnv2, line)...)
		case ".py":
			matchedKeys = append(matchedKeys, extractRegexMatches(rePyEnv1, line)...)
			matchedKeys = append(matchedKeys, extractRegexMatches(rePyEnv2, line)...)
			matchedKeys = append(matchedKeys, extractRegexMatches(rePyEnv3, line)...)
		case ".rs":
			matchedKeys = append(matchedKeys, extractRegexMatches(reRustEnv, line)...)
		case ".java", ".kt":
			matchedKeys = append(matchedKeys, extractRegexMatches(reJavaEnv, line)...)
		case ".php":
			matchedKeys = append(matchedKeys, extractRegexMatches(rePhpEnv, line)...)
		case ".rb":
			matchedKeys = append(matchedKeys, extractRegexMatches(reRubyEnv, line)...)
		}

		if len(matchedKeys) > 0 {
			mu.Lock()
			for _, k := range matchedKeys {
				if isEnvVarEligible(k) {
					detectedVars[k] = append(detectedVars[k], EnvUsageOccurrence{
						FilePath: filePath,
						LineNum:  lineNum,
					})
				}
			}
			mu.Unlock()
		}

		lineNum++
	}
}

func extractRegexMatches(re *regexp.Regexp, line string) []string {
	var res []string
	matches := re.FindAllStringSubmatch(line, -1)
	for _, m := range matches {
		if len(m) > 1 && m[1] != "" {
			res = append(res, m[1])
		}
	}
	return res
}

func isEnvVarEligible(k string) bool {
	if k == "NODE_ENV" || k == "PORT" || k == "PATH" || k == "HOME" {
		return true
	}
	if len(k) < 2 {
		return false
	}
	return true
}

func renderEnvAuditBox(report EnvAuditReport) {
	ui.PrintBoxTop()
	fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("ENVIRONMENT VARIABLES AUDIT"))
	ui.PrintBoxDivider2ColWide()

	envLabel := filepath.Base(report.EnvFile)
	if !report.EnvFileFound {
		envLabel = envLabel + " [File Not Found]"
		fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Environment File", color.HiRedString(envLabel))
	} else {
		envStatus := fmt.Sprintf("%s (%d keys found)", envLabel, report.DefinedCount)
		fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Environment File", color.HiYellowString(envStatus))
	}

	targetDir := ui.Truncate(report.TargetDirectory, 35)
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Target Directory", color.HiWhiteString(targetDir))
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Code References", color.HiWhiteString("%d variables detected", report.DetectedCount))

	if report.MissingCount > 0 {
		missingStr := color.HiRedString("%d missing from %s", report.MissingCount, filepath.Base(report.EnvFile))
		fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Missing Definitions", missingStr)
		fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Audit Status", color.HiRedString("❌ Action Required"))
	} else {
		missingStr := color.HiGreenString("0 missing")
		fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Missing Definitions", missingStr)
		fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Audit Status", color.HiGreenString("✅ All Synced"))
	}

	ui.PrintBoxBottom()
}

func fixMissingEnvVars(envPath string, missing []MissingEnvReportItem) {
	f, err := os.OpenFile(envPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		color.Red("Failed to open %s for writing: %v", envPath, err)
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	_, _ = w.WriteString("\n")
	for _, m := range missing {
		_, _ = w.WriteString(fmt.Sprintf("%s=\n", m.Variable))
	}
	_ = w.Flush()
	color.Green("✔ Appended %d missing placeholders to %s", len(missing), envPath)
}
