package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/V3DxNT/Ray/internal/fsutil"
	"github.com/V3DxNT/Ray/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type DuplicateGroup struct {
	Hash     string   `json:"sha256"`
	Size     int64    `json:"file_size"`
	FileSize string   `json:"file_size_formatted"`
	Files    []string `json:"files"`
}

type DedupReport struct {
	TargetDirectory string           `json:"target_directory"`
	TotalFiles      int              `json:"total_files_scanned"`
	DuplicateGroups []DuplicateGroup `json:"duplicate_groups"`
	ReclaimableSize int64            `json:"reclaimable_bytes"`
	Reclaimable     string           `json:"reclaimable_formatted"`
}

var (
	dedupSkipDirs []string
	dedupJson     bool
	dedupMinSize  int64
)

var dedupCmd = &cobra.Command{
	Use:   "dedup [path]",
	Short: "Find identical duplicate files across your project",
	Long:  "Quickly find duplicate copies of files taking up unnecessary space using SHA-256 checksums. It groups identical files together and shows how much storage you can recover.",
	Example: `  ray dedup
  ray dedup --min-size 1024
  ray dedup --skip vendor,build
  ray dedup --json`,
	Run: runDedup,
}

func init() {
	RootCmd.AddCommand(dedupCmd)
	dedupCmd.Flags().StringSliceVarP(&dedupSkipDirs, "skip", "s", []string{}, "Folders or files to skip")
	dedupCmd.Flags().BoolVar(&dedupJson, "json", false, "Output results in JSON format")
	dedupCmd.Flags().Int64VarP(&dedupMinSize, "min-size", "m", 1, "Smallest file size in bytes to check")
}

func runDedup(cmd *cobra.Command, args []string) {
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

	sizeGroups := make(map[int64][]string)
	var mu sync.Mutex
	var totalScanned int

	spinner := ui.NewSpinner("Scanning for duplicates...")
	if !dedupJson {
		fmt.Println()
		fmt.Printf("🔍 Scanning: %s\n\n", color.YellowString(absPath))
		spinner.Start()
	}

	_ = filepath.Walk(absPath, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if f.IsDir() {
			if path != absPath && fsutil.ShouldSkip(f.Name(), dedupSkipDirs) {
				return filepath.SkipDir
			}
			return nil
		}
		if f.Size() < dedupMinSize {
			return nil
		}
		spinner.Increment()
		mu.Lock()
		totalScanned++
		sizeGroups[f.Size()] = append(sizeGroups[f.Size()], path)
		mu.Unlock()
		return nil
	})

	var candidates [][]string
	for _, files := range sizeGroups {
		if len(files) > 1 {
			candidates = append(candidates, files)
		}
	}

	hashMap := make(map[string][]string)
	fileSizes := make(map[string]int64)
	var hashMu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 64)

	for _, group := range candidates {
		for _, fPath := range group {
			wg.Add(1)
			sem <- struct{}{}
			go func(p string) {
				defer wg.Done()
				defer func() { <-sem }()
				h, sz, err := hashFile(p)
				if err == nil && h != "" {
					hashMu.Lock()
					hashMap[h] = append(hashMap[h], p)
					fileSizes[h] = sz
					hashMu.Unlock()
				}
			}(fPath)
		}
	}
	wg.Wait()

	if !dedupJson {
		spinner.Stop()
	}

	var dupGroups []DuplicateGroup
	var totalReclaimable int64

	for h, paths := range hashMap {
		if len(paths) > 1 {
			sz := fileSizes[h]
			reclaim := sz * int64(len(paths)-1)
			totalReclaimable += reclaim
			dupGroups = append(dupGroups, DuplicateGroup{
				Hash:     h,
				Size:     sz,
				FileSize: ui.FormatBytes(sz),
				Files:    paths,
			})
		}
	}

	sort.Slice(dupGroups, func(i, j int) bool {
		return dupGroups[i].Size > dupGroups[j].Size
	})

	report := DedupReport{
		TargetDirectory: absPath,
		TotalFiles:      totalScanned,
		DuplicateGroups: dupGroups,
		ReclaimableSize: totalReclaimable,
		Reclaimable:     ui.FormatBytes(totalReclaimable),
	}

	if dedupJson {
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(data))
		return
	}

	renderDedupBox(report)
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	hasher := sha256.New()
	n, err := io.Copy(hasher, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hasher.Sum(nil)), n, nil
}

func renderDedupBox(report DedupReport) {
	ui.PrintBoxTop()
	fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("IDENTICAL FILE DUPLICATION REPORT"))
	ui.PrintBoxDivider2ColWide()
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Target Directory", color.HiWhiteString(ui.Truncate(report.TargetDirectory, 35)))
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Files Inspected", color.HiWhiteString("%d files", report.TotalFiles))
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Duplicate Sets", color.HiYellowString("%d groups", len(report.DuplicateGroups)))
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Potential Reclaim", color.HiGreenString(report.Reclaimable))
	ui.PrintBoxDivider()

	if len(report.DuplicateGroups) == 0 {
		fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiGreenString("✔ No duplicate files found across repository!"))
		ui.PrintBoxBottom()
		fmt.Println()
		return
	}

	for idx, g := range report.DuplicateGroups {
		shortHash := g.Hash
		if len(shortHash) > 16 {
			shortHash = shortHash[:16] + "..."
		}
		fmt.Printf(color.CyanString("│")+" %s %s (%s each) %-22s"+color.CyanString("│\n"),
			color.YellowString("[%d]", idx+1),
			color.HiCyanString(shortHash),
			color.HiWhiteString(g.FileSize),
			"",
		)
		for _, f := range g.Files {
			rel, err := filepath.Rel(report.TargetDirectory, f)
			if err != nil || rel == "" {
				rel = f
			} else {
				rel = "./" + rel
			}
			fmt.Printf(color.CyanString("│")+"   ├── %-53s "+color.CyanString("│\n"), color.HiBlackString(ui.Truncate(rel, 53)))
		}
		if idx < len(report.DuplicateGroups)-1 {
			ui.PrintBoxDivider()
		}
	}
	ui.PrintBoxBottom()
	fmt.Println()
}
