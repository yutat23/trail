package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
)

// ---------- 色付き表示のための構造体 ----------

type ColorPattern struct {
	Pattern *regexp.Regexp
	Color   *color.Color
	Order   int
}

var colorPatterns []ColorPattern

type colorMode string

const (
	colorAuto   colorMode = "auto"
	colorAlways colorMode = "always"
	colorNever  colorMode = "never"
)

var selectedColorMode = colorAuto

// バージョン情報
const version = "0.1.3"

func newColor(attrs ...color.Attribute) *color.Color {
	c := color.New(attrs...)
	switch selectedColorMode {
	case colorAlways:
		c.EnableColor()
	case colorNever:
		c.DisableColor()
	}
	return c
}

// 色名をcolor.Colorに変換
func getColor(colorName string) (*color.Color, bool) {
	switch strings.ToLower(colorName) {
	case "red":
		return newColor(color.FgRed), true
	case "green":
		return newColor(color.FgGreen), true
	case "blue":
		return newColor(color.FgBlue), true
	case "yellow":
		return newColor(color.FgYellow), true
	case "magenta":
		return newColor(color.FgMagenta), true
	case "cyan":
		return newColor(color.FgCyan), true
	case "white":
		return newColor(color.FgWhite), true
	case "black":
		return newColor(color.FgBlack), true
	case "brightred":
		return newColor(color.FgHiRed), true
	case "brightgreen":
		return newColor(color.FgHiGreen), true
	case "brightblue":
		return newColor(color.FgHiBlue), true
	case "brightyellow":
		return newColor(color.FgHiYellow), true
	case "brightmagenta":
		return newColor(color.FgHiMagenta), true
	case "brightcyan":
		return newColor(color.FgHiCyan), true
	case "brightwhite":
		return newColor(color.FgHiWhite), true
	default:
		return nil, false
	}
}

// 文字列に色付きパターンを適用
func applyColorPatterns(text string) string {
	if len(colorPatterns) == 0 {
		return text
	}

	type colorMatch struct {
		start int
		end   int
		color *color.Color
		order int
	}

	var allMatches []colorMatch
	for _, pattern := range colorPatterns {
		matches := pattern.Pattern.FindAllStringIndex(text, -1)
		for _, match := range matches {
			if match[0] == match[1] {
				continue
			}
			allMatches = append(allMatches, colorMatch{
				start: match[0],
				end:   match[1],
				color: pattern.Color,
				order: pattern.Order,
			})
		}
	}
	if len(allMatches) == 0 {
		return text
	}

	// 重複する場合は、後から指定されたパターンを優先する。
	sort.SliceStable(allMatches, func(i, j int) bool {
		if allMatches[i].order != allMatches[j].order {
			return allMatches[i].order > allMatches[j].order
		}
		if allMatches[i].end-allMatches[i].start != allMatches[j].end-allMatches[j].start {
			return allMatches[i].end-allMatches[i].start > allMatches[j].end-allMatches[j].start
		}
		return allMatches[i].start < allMatches[j].start
	})

	var finalMatches []colorMatch
	for _, match := range allMatches {
		overlaps := false
		for _, existing := range finalMatches {
			if (match.start >= existing.start && match.start < existing.end) ||
				(match.end > existing.start && match.end <= existing.end) ||
				(match.start <= existing.start && match.end >= existing.end) {
				overlaps = true
				break
			}
		}
		if !overlaps {
			finalMatches = append(finalMatches, match)
		}
	}
	sort.Slice(finalMatches, func(i, j int) bool {
		return finalMatches[i].start < finalMatches[j].start
	})

	var result strings.Builder
	lastEnd := 0
	for _, match := range finalMatches {
		result.WriteString(text[lastEnd:match.start])
		result.WriteString(match.color.Sprint(text[match.start:match.end]))
		lastEnd = match.end
	}
	result.WriteString(text[lastEnd:])

	return result.String()
}

// 色付きパターンを解析
func parseColorPatterns(colorOpts []string) {
	for _, colorOpt := range colorOpts {
		patterns := strings.Split(colorOpt, ",")
		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}

			parts := strings.SplitN(pattern, ":", 2)
			if len(parts) != 2 {
				log.Printf("invalid color pattern format: %s (expected 'color:regex')", pattern)
				continue
			}

			colorName := strings.TrimSpace(parts[0])
			regexStr := strings.TrimSpace(parts[1])
			if regexStr == "" {
				log.Printf("empty regex pattern in: %s", pattern)
				continue
			}

			regex, err := regexp.Compile(regexStr)
			if err != nil {
				log.Printf("invalid regex pattern '%s': %v", regexStr, err)
				continue
			}

			colorValue, ok := getColor(colorName)
			if !ok {
				log.Printf("invalid color name '%s'", colorName)
				continue
			}

			colorPatterns = append(colorPatterns, ColorPattern{
				Pattern: regex,
				Color:   colorValue,
				Order:   len(colorPatterns),
			})
		}
	}
}

type repeatedStrings []string

func (r *repeatedStrings) String() string {
	if r == nil {
		return ""
	}
	return strings.Join(*r, ",")
}

func (r *repeatedStrings) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*r = append(*r, value)
	return nil
}

// ---------- 共通ヘルパ ----------

type globalOptions struct {
	showLogo  bool
	colorLogo bool
}

func parseGlobalArgs(args []string) (globalOptions, string, []string) {
	opts := globalOptions{showLogo: true, colorLogo: true}
	for len(args) > 0 {
		arg := args[0]
		switch {
		case arg == "--no-logo":
			opts.showLogo = false
			args = args[1:]
		case arg == "--no-color-logo":
			opts.colorLogo = false
			args = args[1:]
		case arg == "--version" || arg == "-v":
			fmt.Println(version)
			os.Exit(0)
		case arg == "--color":
			if len(args) < 2 {
				log.Fatal("missing value for --color (auto, always, never)")
			}
			setColorMode(args[1])
			args = args[2:]
		case strings.HasPrefix(arg, "--color="):
			setColorMode(strings.TrimPrefix(arg, "--color="))
			args = args[1:]
		case arg == "-h" || arg == "--help" || arg == "help":
			usage(opts, 0)
		default:
			return opts, arg, args[1:]
		}
	}
	return opts, "", nil
}

func setColorMode(mode string) {
	switch strings.ToLower(mode) {
	case "auto":
		selectedColorMode = colorAuto
		return
	case "always":
		selectedColorMode = colorAlways
		color.NoColor = false
	case "never":
		selectedColorMode = colorNever
		color.NoColor = true
	default:
		log.Fatalf("invalid --color value %q (expected auto, always, never)", mode)
	}
}

func applyColorOptions(colorOpts repeatedStrings) {
	if len(colorOpts) == 0 {
		return
	}
	parseColorPatterns(colorOpts)
}

func validateLineCount(n int) {
	if n < 0 {
		log.Fatalf("-n must be >= 0")
	}
}

func validateInterval(interval time.Duration) {
	if interval <= 0 {
		log.Fatalf("-interval must be > 0")
	}
}

func printLine(text string) {
	fmt.Println(applyColorPatterns(text))
}

// 最新 (mod time が最大) の通常ファイルを返す
func newestFile(dir string) (string, error) {
	return newestFileWithPattern(dir, "*")
}

// ワイルドカードパターンにマッチする最新のファイルを返す
func newestFileWithPattern(dir, pattern string) (string, error) {
	var newest string
	var newestMod time.Time

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return "", fmt.Errorf("invalid pattern '%s': %v", pattern, err)
		}
		if !matched {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return "", err
		}
		if info.Mode().IsRegular() && info.ModTime().After(newestMod) {
			newest = filepath.Join(dir, entry.Name())
			newestMod = info.ModTime()
		}
	}
	if newest == "" {
		return "", fmt.Errorf("no files matching pattern '%s' in %s", pattern, dir)
	}
	return newest, nil
}

// ファイルを tail して標準出力へ
func startFollow(path string, offset int64) (*tail.Tail, <-chan error, error) {
	cfg := tail.Config{
		Follow:    true,
		ReOpen:    true, // ローテーション追従
		MustExist: true,
		Poll:      true, // cross-platform
		Logger:    tail.DiscardingLogger,
		Location:  &tail.SeekInfo{Offset: offset, Whence: io.SeekStart},
	}
	t, err := tail.TailFile(path, cfg)
	if err != nil {
		return nil, nil, err
	}

	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		for line := range t.Lines {
			if line.Err != nil {
				errCh <- line.Err
				return
			}
			printLine(line.Text)
		}
	}()
	return t, errCh, nil
}

// ---------- サブコマンド: file ----------

func cmdFile(args []string) {
	fs := flag.NewFlagSet("file", flag.ExitOnError)
	nLines := fs.Int("n", 10, "show last N lines then follow")
	var colorOpts repeatedStrings
	fs.Var(&colorOpts, "c", "color patterns in format 'color:regex' (can be used multiple times)")
	fs.Parse(args)

	if fs.NArg() != 1 {
		log.Fatalf("usage: trail file [options] <file>")
	}
	validateLineCount(*nLines)
	file := fs.Arg(0)

	applyColorOptions(colorOpts)

	offset, err := printLastN(file, *nLines)
	if err != nil {
		log.Fatal(err)
	}

	_, errCh, err := startFollow(file, offset)
	if err != nil {
		log.Fatal(err)
	}
	if err := <-errCh; err != nil {
		log.Fatal(err)
	}
}

func printLastN(path string, n int) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	if n == 0 {
		offset, err := f.Seek(0, io.SeekEnd)
		return offset, err
	}

	ring := make([]string, n)
	count := 0
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			ring[count%n] = line
			count++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	start := 0
	if count > n {
		start = count - n
	}
	for i := start; i < count; i++ {
		printLine(ring[i%n])
	}

	offset, err := f.Seek(0, io.SeekCurrent)
	return offset, err
}

// ---------- サブコマンド: dir ----------

func cmdDir(args []string) {
	fs := flag.NewFlagSet("dir", flag.ExitOnError)
	interval := fs.Duration("interval", 5*time.Second, "fallback polling interval")
	var colorOpts repeatedStrings
	fs.Var(&colorOpts, "c", "color patterns in format 'color:regex' (can be used multiple times)")
	pattern := fs.String("pattern", "*", "file pattern to match (e.g., '*.log', 'app-*.log')")
	nLines := fs.Int("n", 10, "show last N lines then follow")
	fs.Parse(args)
	if fs.NArg() != 1 {
		log.Fatalf("usage: trail dir [options] <directory>")
	}
	validateLineCount(*nLines)
	validateInterval(*interval)
	dir := fs.Arg(0)

	applyColorOptions(colorOpts)

	current, err := newestFileWithPattern(dir, *pattern)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("trailing %s (pattern: %s)", current, *pattern)

	offset, err := printLastN(current, *nLines)
	if err != nil {
		log.Fatal(err)
	}

	currentTail, currentErrCh, err := startFollow(current, offset)
	if err != nil {
		log.Fatal(err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	if err := watcher.Add(dir); err != nil {
		log.Fatal(err)
	}

	timer := time.NewTicker(*interval)
	defer timer.Stop()

	switchToLatest := func() {
		latest, err := newestFileWithPattern(dir, *pattern)
		if err != nil {
			log.Printf("latest file check failed: %v", err)
			return
		}
		if latest == current {
			return
		}
		if currentTail != nil {
			if err := currentTail.Stop(); err != nil {
				log.Printf("failed to stop tail for %s: %v", current, err)
			}
			if currentErrCh != nil {
				<-currentErrCh
			}
			currentTail.Cleanup()
		}

		current = latest
		log.Printf("switching to %s", current)
		offset, err := printLastN(current, *nLines)
		if err != nil {
			log.Printf("failed to print last lines for %s: %v", current, err)
			return
		}
		currentTail, currentErrCh, err = startFollow(current, offset)
		if err != nil {
			log.Printf("failed to follow %s: %v", current, err)
		}
	}

	for {
		select {
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
				switchToLatest()
			}
		case <-timer.C:
			switchToLatest()
		case err, ok := <-currentErrCh:
			if !ok {
				currentErrCh = nil
				continue
			}
			if err != nil {
				log.Printf("tail error for %s: %v", current, err)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watch error: %v", err)
		}
	}
}

// ---------- ロゴ表示 ----------

func showLogo(w io.Writer, colored bool) {
	if !colored {
		showSimpleLogo(w)
		return
	}
	if err := showColoredLogo(w); err != nil {
		showSimpleLogo(w)
	}
}

func showSimpleLogo(w io.Writer) {
	logo := `
████████╗██████╗  █████╗ ██╗██╗
╚══██╔══╝██╔══██╗██╔══██╗██║██║
   ██║   ██████╔╝███████║██║██║
   ██║   ██╔══██╗██╔══██║██║██║
   ██║   ██║  ██║██║  ██║██║███████╗
   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝╚══════╝

   Tail with log-rotate follow
   Version ` + version + `
`
	fmt.Fprint(w, logo)
}

func showColoredLogo(w io.Writer) error {
	colors := []*color.Color{
		newColor(color.FgHiBlue),
		newColor(color.FgHiCyan),
		newColor(color.FgHiGreen),
		newColor(color.FgHiYellow),
		newColor(color.FgHiRed),
		newColor(color.FgHiMagenta),
	}
	logoLines := []string{
		"████████╗██████╗  █████╗ ██╗██╗     ",
		"╚══██╔══╝██╔══██╗██╔══██╗██║██║     ",
		"   ██║   ██████╔╝███████║██║██║     ",
		"   ██║   ██╔══██╗██╔══██║██║██║     ",
		"   ██║   ██║  ██║██║  ██║██║███████╗",
		"   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝╚══════╝",
		"",
		"   Tail with log-rotate follow",
		"   Version " + version,
	}

	for i, line := range logoLines {
		if line == "" {
			fmt.Fprintln(w)
			continue
		}
		colorIndex := i % len(colors)
		fmt.Fprintln(w, colors[colorIndex].Sprint(line))
	}

	return nil
}

// ---------- main ----------

func main() {
	opts, command, args := parseGlobalArgs(os.Args[1:])
	if command == "" {
		usage(opts, 1)
	}
	switch command {
	case "-f", "file":
		cmdFile(args)
	case "-d", "dir":
		cmdDir(args)
	case "-h", "--help", "help":
		usage(opts, 0)
	default:
		log.Fatalf("unknown command %q\n\n", command)
	}
}

func usage(opts globalOptions, exitCode int) {
	w := io.Writer(os.Stdout)
	if exitCode != 0 {
		w = os.Stderr
	}
	if opts.showLogo {
		showLogo(w, opts.colorLogo)
	}
	fmt.Fprintf(w, `trail - tail with log-rotate follow

USAGE
  trail [options] <command> [options] <path>
COMMANDS
  -f, file       Tail a file and follow it
  -d, dir        Tail the latest file in a directory and follow it

COMMON OPTIONS
  -h, --help         Show this help
  -v, --version      Show version
  --no-logo          Disable logo display
  --no-color-logo    Disable colored logo (use simple ASCII art)
  --color <mode>     Color output mode: auto, always, never (default auto)

file OPTIONS
  -n <N>         Print last N lines before following (default 10)
  -c <patterns>  Color patterns in format 'color:regex' (can be used multiple times)
                 Colors: red, green, blue, yellow, magenta, cyan, white, black
                 Bright colors: brightred, brightgreen, brightblue, brightyellow, brightmagenta, brightcyan, brightwhite

dir  OPTIONS
  -n <N>         Print last N lines before following (default 10)
  -interval <d>  Polling fallback interval (default 5s)
  -c <patterns>  Color patterns in format 'color:regex' (can be used multiple times)
  -pattern <p>   File pattern to match (e.g., '*.log', 'app-*.log', 'service-*.txt')

EXAMPLES
  trail file -n 100 app.log
  trail dir  "C:\Logs\MyService"
  trail dir -n 20 "C:\Logs\MyService"
  trail dir -pattern "*.log" "C:\Logs\MyService"
  trail dir -pattern "app-*.log" -n 50 "C:\Logs\MyService"
  trail file -c "red:ERROR,green:DEBUG,blue:\d{2}-\d{2}" app.log
  trail file -c "red:ERROR" -c "green:DEBUG" app.log
  trail dir -c "yellow:WARN,red:ERROR" "C:\Logs\MyService"
  trail dir -pattern "*.log" -c "red:ERROR" "C:\Logs\MyService"
  trail --no-logo file app.log
  trail --no-color-logo file app.log
  trail --color always file -c "red:ERROR" app.log
`)
	os.Exit(exitCode)
}
