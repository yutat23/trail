package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
)

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
		fmt.Println(line.Text)
	}
	return nil
}

// ---------- サブコマンド: file ----------

func cmdFile(args []string) {
	fs := flag.NewFlagSet("file", flag.ExitOnError)
	nLines := fs.Int("n", 10, "show last N lines then follow")
	fs.Parse(args)

	if fs.NArg() != 1 {
		log.Fatalf("usage: trail file [options] <file>")
	}
	file := fs.Arg(0)

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
		fmt.Println(l)
	}
}

// ---------- サブコマンド: dir ----------

func cmdDir(args []string) {
	fs := flag.NewFlagSet("dir", flag.ExitOnError)
	interval := fs.Duration("interval", 5*time.Second, "fallback polling interval")
	fs.Parse(args)
	if fs.NArg() != 1 {
		log.Fatalf("usage: trail dir [options] <directory>")
	}
	dir := fs.Arg(0)

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

dir  OPTIONS
  -interval <d>  Polling fallback interval (default 5s)

EXAMPLES
  trail file -n 100 app.log
  trail dir  "C:\Logs\MyService"
`)
	os.Exit(1)
}
