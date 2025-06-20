package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
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
}

var colorPatterns []ColorPattern

// 色名をcolor.Colorに変換
func getColor(colorName string) *color.Color {
	switch strings.ToLower(colorName) {
	case "red":
		return color.New(color.FgRed)
	case "green":
		return color.New(color.FgGreen)
	case "blue":
		return color.New(color.FgBlue)
	case "yellow":
		return color.New(color.FgYellow)
	case "magenta":
		return color.New(color.FgMagenta)
	case "cyan":
		return color.New(color.FgCyan)
	case "white":
		return color.New(color.FgWhite)
	case "black":
		return color.New(color.FgBlack)
	case "brightred":
		return color.New(color.FgHiRed)
	case "brightgreen":
		return color.New(color.FgHiGreen)
	case "brightblue":
		return color.New(color.FgHiBlue)
	case "brightyellow":
		return color.New(color.FgHiYellow)
	case "brightmagenta":
		return color.New(color.FgHiMagenta)
	case "brightcyan":
		return color.New(color.FgHiCyan)
	case "brightwhite":
		return color.New(color.FgHiWhite)
	default:
		return color.New(color.FgWhite) // デフォルトは白色
	}
}

// 文字列に色付きパターンを適用
func applyColorPatterns(text string) string {
	result := text
	for _, pattern := range colorPatterns {
		matches := pattern.Pattern.FindAllStringIndex(text, -1)
		// 後ろから処理してインデックスがずれないようにする
		for i := len(matches) - 1; i >= 0; i-- {
			match := matches[i]
			matchedText := text[match[0]:match[1]]
			coloredText := pattern.Color.Sprint(matchedText)
			result = result[:match[0]] + coloredText + result[match[1]:]
		}
	}
	return result
}

// 色付きパターンを解析
func parseColorPatterns(colorOpts string) {
	patterns := strings.Split(colorOpts, ",")
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		
		// "color:regex" の形式で解析
		parts := strings.SplitN(pattern, ":", 2)
		if len(parts) != 2 {
			log.Printf("invalid color pattern format: %s (expected 'color:regex')", pattern)
			continue
		}
		
		colorName := strings.TrimSpace(parts[0])
		regexStr := strings.TrimSpace(parts[1])
		
		// 正規表現をコンパイル
		regex, err := regexp.Compile(regexStr)
		if err != nil {
			log.Printf("invalid regex pattern '%s': %v", regexStr, err)
			continue
		}
		
		// 色を取得
		color := getColor(colorName)
		
		// パターンを追加
		colorPatterns = append(colorPatterns, ColorPattern{
			Pattern: regex,
			Color:   color,
		})
	}
}

// ---------- 共通ヘルパ ----------

// 最新 (mod time が最大) の通常ファイルを返す
func newestFile(dir string) (string, error) {
	var newest string
	var newestMod time.Time

	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && info.ModTime().After(newestMod) {
			newest, newestMod = p, info.ModTime()
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if newest == "" {
		return "", fmt.Errorf("no regular files in %s", dir)
	}
	return newest, nil
}

// ファイルを tail して標準出力へ
func follow(path string) error {
	cfg := tail.Config{
		Follow:    true,
		ReOpen:    true, // ローテーション追従
		MustExist: true,
		Poll:      true, // cross-platform
		Logger:    tail.DiscardingLogger,
	}
	t, err := tail.TailFile(path, cfg)
	if err != nil {
		return err
	}
	for line := range t.Lines {
		coloredLine := applyColorPatterns(line.Text)
		fmt.Println(coloredLine)
	}
	return nil
}

// ---------- サブコマンド: file ----------

func cmdFile(args []string) {
	fs := flag.NewFlagSet("file", flag.ExitOnError)
	nLines := fs.Int("n", 10, "show last N lines then follow")
	colorOpts := fs.String("c", "", "color patterns in format 'color:regex' (can be used multiple times)")
	fs.Parse(args)

	if fs.NArg() != 1 {
		log.Fatalf("usage: trail file [options] <file>")
	}
	file := fs.Arg(0)

	// 色付きパターンを解析
	if *colorOpts != "" {
		parseColorPatterns(*colorOpts)
	}

	// 直近 N 行だけ先に出力
	printLastN(file, *nLines)

	if err := follow(file); err != nil {
		log.Fatal(err)
	}
}

func printLastN(path string, n int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	all := strings.Split(string(data), "\n")
	if n > len(all) {
		n = len(all)
	}
	for _, l := range all[len(all)-n:] {
		coloredLine := applyColorPatterns(l)
		fmt.Println(coloredLine)
	}
}

// ---------- サブコマンド: dir ----------

func cmdDir(args []string) {
	fs := flag.NewFlagSet("dir", flag.ExitOnError)
	interval := fs.Duration("interval", 5*time.Second, "fallback polling interval")
	colorOpts := fs.String("c", "", "color patterns in format 'color:regex' (can be used multiple times)")
	fs.Parse(args)
	if fs.NArg() != 1 {
		log.Fatalf("usage: trail dir [options] <directory>")
	}
	dir := fs.Arg(0)

	// 色付きパターンを解析
	if *colorOpts != "" {
		parseColorPatterns(*colorOpts)
	}

	// 最初の対象ファイル
	current, err := newestFile(dir)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("trailing %s", current)
	go func() {
		if err := follow(current); err != nil {
			log.Fatal(err)
		}
	}()

	// fsnotify で新規ファイル誕生を監視
	watcher, _ := fsnotify.NewWatcher()
	defer watcher.Close()
	_ = watcher.Add(dir)

	timer := time.NewTicker(*interval)
	for {
		select {
		case ev := <-watcher.Events:
			if ev.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
				if latest, _ := newestFile(dir); latest != current {
					current = latest
					log.Printf("switching to %s", current)
					go follow(current)
				}
			}
		case <-timer.C: // 監視失敗時の保険
			if latest, _ := newestFile(dir); latest != current {
				current = latest
				log.Printf("switching to %s", current)
				go follow(current)
			}
		case err := <-watcher.Errors:
			log.Printf("watch error: %v", err)
		}
	}
}

// ---------- main ----------

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "-f", "file":
		cmdFile(os.Args[2:])
	case "-d", "dir":
		cmdDir(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		log.Fatalf("unknown command %q\n\n", os.Args[1])
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `trail - tail with log-rotate follow

USAGE
	trail <command> [options] <path>
COMMANDS
	-f, file       Tail a file and follow it
	-d, dir        Tail the latest file in a directory and follow it

COMMON OPTIONS
  -h, --help     Show this help

file OPTIONS
  -n <N>         Print last N lines before following (default 10)
  -c <patterns>  Color patterns in format 'color:regex' (can be used multiple times)
                 Colors: red, green, blue, yellow, magenta, cyan, white, black
                 Bright colors: brightred, brightgreen, brightblue, brightyellow, brightmagenta, brightcyan, brightwhite

dir  OPTIONS
  -interval <d>  Polling fallback interval (default 5s)
  -c <patterns>  Color patterns in format 'color:regex' (can be used multiple times)

EXAMPLES
  trail file -n 100 app.log
  trail dir  "C:\Logs\MyService"
  trail file -c "red:ERROR,green:DEBUG,blue:\d{2}-\d{2}" app.log
  trail dir -c "yellow:WARN,red:ERROR" "C:\Logs\MyService"
`)
	os.Exit(1)
}
