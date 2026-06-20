package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func resetTestState() {
	colorPatterns = nil
	selectedColorMode = colorAuto
	color.NoColor = true
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("")
}

func withReset(t *testing.T) {
	t.Helper()
	resetTestState()
	t.Cleanup(resetTestState)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdout = w
	closed := false
	defer func() {
		os.Stdout = oldStdout
		if !closed {
			_ = w.Close()
			_ = r.Close()
		}
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	closed = true

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func captureLogOutput(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	oldOutput := log.Writer()
	oldFlags := log.Flags()
	oldPrefix := log.Prefix()
	log.SetOutput(&buf)
	log.SetFlags(0)
	log.SetPrefix("")
	defer func() {
		log.SetOutput(oldOutput)
		log.SetFlags(oldFlags)
		log.SetPrefix(oldPrefix)
	}()

	fn()
	return buf.String()
}

func ansi(code, text string) string {
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func requireContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("got %q, want it to contain %q", got, want)
	}
}

func requireNotContains(t *testing.T, got, want string) {
	t.Helper()
	if strings.Contains(got, want) {
		t.Fatalf("got %q, want it not to contain %q", got, want)
	}
}

func writeFileAt(t *testing.T, path, content string, modTime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}

func appendToFile(t *testing.T, path, content string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
}

func TestGetColorSupportsDocumentedNames(t *testing.T) {
	withReset(t)
	setColorMode("always")

	tests := []struct {
		name string
		code string
	}{
		{"red", "31"},
		{"green", "32"},
		{"blue", "34"},
		{"yellow", "33"},
		{"magenta", "35"},
		{"cyan", "36"},
		{"white", "37"},
		{"black", "30"},
		{"brightred", "91"},
		{"brightgreen", "92"},
		{"brightblue", "94"},
		{"brightyellow", "93"},
		{"brightmagenta", "95"},
		{"brightcyan", "96"},
		{"brightwhite", "97"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, ok := getColor(strings.ToUpper(tt.name))
			if !ok {
				t.Fatalf("getColor(%q) returned ok=false", tt.name)
			}
			if got, want := c.Sprint("x"), ansi(tt.code, "x"); got != want {
				t.Fatalf("getColor(%q).Sprint = %q, want %q", tt.name, got, want)
			}
		})
	}

	if c, ok := getColor("orange"); ok || c != nil {
		t.Fatalf("getColor invalid = (%v, %v), want (nil, false)", c, ok)
	}
}

func TestColorModesControlANSIEscapes(t *testing.T) {
	t.Run("auto respects disabled color", func(t *testing.T) {
		withReset(t)
		color.NoColor = true
		parseColorPatterns([]string{"red:ERROR"})

		if got := applyColorPatterns("ERROR"); got != "ERROR" {
			t.Fatalf("auto color output = %q, want plain text", got)
		}
	})

	t.Run("always emits ANSI", func(t *testing.T) {
		withReset(t)
		setColorMode("always")
		parseColorPatterns([]string{"red:ERROR"})

		if got, want := applyColorPatterns("ERROR"), ansi("31", "ERROR"); got != want {
			t.Fatalf("always color output = %q, want %q", got, want)
		}
	})

	t.Run("never suppresses ANSI", func(t *testing.T) {
		withReset(t)
		setColorMode("never")
		parseColorPatterns([]string{"red:ERROR"})

		if got := applyColorPatterns("ERROR"); got != "ERROR" {
			t.Fatalf("never color output = %q, want plain text", got)
		}
	})
}

func TestParseColorPatternsTrimsMultipleOptionsAndKeepsOrder(t *testing.T) {
	withReset(t)
	setColorMode("always")

	parseColorPatterns([]string{" red:ERROR , green:DEBUG ", "cyan:日本語"})

	if got, want := len(colorPatterns), 3; got != want {
		t.Fatalf("len(colorPatterns) = %d, want %d", got, want)
	}
	for i, pattern := range colorPatterns {
		if pattern.Order != i {
			t.Fatalf("pattern %d order = %d, want %d", i, pattern.Order, i)
		}
	}

	got := applyColorPatterns("ERROR DEBUG 日本語")
	want := ansi("31", "ERROR") + " " + ansi("32", "DEBUG") + " " + ansi("36", "日本語")
	if got != want {
		t.Fatalf("colored output = %q, want %q", got, want)
	}
}

func TestParseColorPatternsLogsInvalidEntries(t *testing.T) {
	withReset(t)

	logs := captureLogOutput(t, func() {
		parseColorPatterns([]string{
			"orange:ERROR, blue:, yellow:[",
			"magenta:成功",
			"bad-format",
		})
	})

	if got, want := len(colorPatterns), 1; got != want {
		t.Fatalf("len(colorPatterns) = %d, want %d", got, want)
	}
	if got := colorPatterns[0].Pattern.String(); got != "成功" {
		t.Fatalf("valid pattern = %q, want %q", got, "成功")
	}

	for _, want := range []string{
		"invalid color name 'orange'",
		"empty regex pattern in: blue:",
		"invalid regex pattern '['",
		"invalid color pattern format: bad-format",
	} {
		requireContains(t, logs, want)
	}
}

func TestApplyColorPatterns(t *testing.T) {
	t.Run("no patterns returns original text", func(t *testing.T) {
		withReset(t)

		got := applyColorPatterns("2026-06-21 INFO 起動しました")
		if got != "2026-06-21 INFO 起動しました" {
			t.Fatalf("applyColorPatterns = %q", got)
		}
	})

	t.Run("no matches returns original text", func(t *testing.T) {
		withReset(t)
		setColorMode("always")
		parseColorPatterns([]string{"red:ERROR"})

		got := applyColorPatterns("2026-06-21 INFO 起動しました")
		if got != "2026-06-21 INFO 起動しました" {
			t.Fatalf("applyColorPatterns = %q", got)
		}
	})

	t.Run("colors adjacent English and Japanese matches", func(t *testing.T) {
		withReset(t)
		setColorMode("always")
		parseColorPatterns([]string{"red:ERROR", "yellow:WARN", "cyan:ユーザー[0-9]+"})

		got := applyColorPatterns("ERROR WARN ユーザー123 正常")
		want := ansi("31", "ERROR") + " " + ansi("33", "WARN") + " " + ansi("36", "ユーザー123") + " 正常"
		if got != want {
			t.Fatalf("applyColorPatterns = %q, want %q", got, want)
		}
	})

	t.Run("later overlapping pattern wins even when narrower", func(t *testing.T) {
		withReset(t)
		setColorMode("always")
		parseColorPatterns([]string{"red:ERROR 詳細", "green:ERROR"})

		got := applyColorPatterns("ERROR 詳細")
		want := ansi("32", "ERROR") + " 詳細"
		if got != want {
			t.Fatalf("applyColorPatterns = %q, want %q", got, want)
		}
	})

	t.Run("ignores zero length regexp matches", func(t *testing.T) {
		withReset(t)
		setColorMode("always")
		parseColorPatterns([]string{"red:^", "green:INFO"})

		got := applyColorPatterns("INFO")
		want := ansi("32", "INFO")
		if got != want {
			t.Fatalf("applyColorPatterns = %q, want %q", got, want)
		}
	})
}

func TestApplyColorPatternsDoesNotCorruptJapaneseLogs(t *testing.T) {
	withReset(t)
	setColorMode("always")
	parseColorPatterns([]string{
		"brightcyan:ユーザー登録",
		"brightred:失敗しました",
		"green:処理が完了しました",
	})

	text := "2026-06-21 10:00:02 ERROR ユーザー登録に失敗しました"
	got := applyColorPatterns(text)

	requireContains(t, got, ansi("96", "ユーザー登録"))
	requireContains(t, got, ansi("91", "失敗しました"))
	requireNotContains(t, got, "\uFFFD")
	if plain := stripANSI(got); plain != text {
		t.Fatalf("stripANSI(output) = %q, want %q", plain, text)
	}
}

func TestPrintLineAppliesColorsToJapaneseText(t *testing.T) {
	withReset(t)
	setColorMode("always")
	parseColorPatterns([]string{"yellow:注意"})

	out := captureStdout(t, func() {
		printLine("WARN 注意してください")
	})

	want := "WARN " + ansi("33", "注意") + "してください\n"
	if out != want {
		t.Fatalf("printLine output = %q, want %q", out, want)
	}
}

func TestPrintLastNWithTrailingNewline(t *testing.T) {
	withReset(t)

	path := filepath.Join(t.TempDir(), "app.log")
	content := "one\ntwo\nthree\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var offset int64
	out := captureStdout(t, func() {
		var err error
		offset, err = printLastN(path, 3)
		if err != nil {
			t.Fatal(err)
		}
	})

	if out != content {
		t.Fatalf("printLastN output = %q, want %q", out, content)
	}
	if offset != int64(len(content)) {
		t.Fatalf("offset = %d, want %d", offset, len(content))
	}
}

func TestPrintLastNHandlesJapaneseCRLFWithoutTrailingNewline(t *testing.T) {
	withReset(t)

	path := filepath.Join(t.TempDir(), "jp.log")
	lines := []string{
		"2026-06-21 10:00:00 INFO 起動しました",
		"2026-06-21 10:00:01 WARN 接続が遅延しています",
		"2026-06-21 10:00:02 ERROR ユーザー登録に失敗しました",
		"2026-06-21 10:00:03 INFO 処理が完了しました",
	}
	content := strings.Join(lines, "\r\n")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var offset int64
	out := captureStdout(t, func() {
		var err error
		offset, err = printLastN(path, 2)
		if err != nil {
			t.Fatal(err)
		}
	})

	want := lines[2] + "\n" + lines[3] + "\n"
	if out != want {
		t.Fatalf("printLastN output = %q, want %q", out, want)
	}
	if offset != int64(len([]byte(content))) {
		t.Fatalf("offset = %d, want %d", offset, len([]byte(content)))
	}
}

func TestPrintLastNVariants(t *testing.T) {
	t.Run("n larger than file prints all lines", func(t *testing.T) {
		withReset(t)

		path := filepath.Join(t.TempDir(), "small.log")
		content := "alpha\nbeta\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		out := captureStdout(t, func() {
			offset, err := printLastN(path, 10)
			if err != nil {
				t.Fatal(err)
			}
			if offset != int64(len(content)) {
				t.Fatalf("offset = %d, want %d", offset, len(content))
			}
		})
		if out != content {
			t.Fatalf("printLastN output = %q, want %q", out, content)
		}
	})

	t.Run("n zero prints nothing and seeks to end", func(t *testing.T) {
		withReset(t)

		path := filepath.Join(t.TempDir(), "zero.log")
		content := "alpha\nbeta\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		out := captureStdout(t, func() {
			offset, err := printLastN(path, 0)
			if err != nil {
				t.Fatal(err)
			}
			if offset != int64(len(content)) {
				t.Fatalf("offset = %d, want %d", offset, len(content))
			}
		})
		if out != "" {
			t.Fatalf("printLastN output = %q, want empty", out)
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		withReset(t)

		out := captureStdout(t, func() {
			if _, err := printLastN(filepath.Join(t.TempDir(), "missing.log"), 1); err == nil {
				t.Fatal("printLastN missing file error = nil")
			}
		})
		if out != "" {
			t.Fatalf("printLastN output = %q, want empty", out)
		}
	})
}

func TestStartFollowPrintsAppendedJapaneseLinesWithColors(t *testing.T) {
	withReset(t)
	setColorMode("always")
	parseColorPatterns([]string{"red:ERROR", "cyan:成功"})

	path := filepath.Join(t.TempDir(), "follow.log")
	initial := "2026-06-21 10:00:00 INFO 既存行\n"
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		tailed, errCh, err := startFollow(path, int64(len([]byte(initial))))
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(100 * time.Millisecond)
		appendToFile(t, path, "2026-06-21 10:00:01 ERROR 失敗しました\n")
		appendToFile(t, path, "2026-06-21 10:00:02 INFO 成功しました\n")
		time.Sleep(1200 * time.Millisecond)

		if err := tailed.Stop(); err != nil {
			t.Fatal(err)
		}
		select {
		case err, ok := <-errCh:
			if ok && err != nil {
				t.Fatal(err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("tail did not stop")
		}
		tailed.Cleanup()
	})

	requireNotContains(t, out, "既存行")
	requireContains(t, out, ansi("31", "ERROR"))
	requireContains(t, out, ansi("36", "成功"))
	requireContains(t, stripANSI(out), "2026-06-21 10:00:01 ERROR 失敗しました\n")
	requireContains(t, stripANSI(out), "2026-06-21 10:00:02 INFO 成功しました\n")
}

func TestNewestFileWithPatternSelectsNewestMatchingRegularFile(t *testing.T) {
	dir := t.TempDir()
	base := time.Now().Add(-1 * time.Hour).Truncate(time.Second)

	oldLog := filepath.Join(dir, "app-1.log")
	newLog := filepath.Join(dir, "app-2.log")
	newerNonMatch := filepath.Join(dir, "notes.txt")
	matchingDir := filepath.Join(dir, "app-3.log")

	writeFileAt(t, oldLog, "old", base)
	writeFileAt(t, newLog, "new", base.Add(1*time.Minute))
	writeFileAt(t, newerNonMatch, "newer but not log", base.Add(2*time.Minute))
	if err := os.Mkdir(matchingDir, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := newestFileWithPattern(dir, "app-*.log")
	if err != nil {
		t.Fatal(err)
	}
	if got != newLog {
		t.Fatalf("newestFileWithPattern = %q, want %q", got, newLog)
	}

	got, err = newestFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != newerNonMatch {
		t.Fatalf("newestFile = %q, want %q", got, newerNonMatch)
	}
}

func TestNewestFileWithPatternUsesLiteralGlobMatching(t *testing.T) {
	dir := t.TempDir()
	oldFile := filepath.Join(dir, "app+1.log")
	newFile := filepath.Join(dir, "app+2.log")
	baseTime := time.Now().Add(-1 * time.Hour).Truncate(time.Second)

	writeFileAt(t, oldFile, "old", baseTime)
	writeFileAt(t, newFile, "new", baseTime.Add(time.Minute))

	got, err := newestFileWithPattern(dir, "app+*.log")
	if err != nil {
		t.Fatal(err)
	}
	if got != newFile {
		t.Fatalf("newestFileWithPattern = %q, want %q", got, newFile)
	}
}

func TestNewestFileWithPatternErrors(t *testing.T) {
	t.Run("no matching file", func(t *testing.T) {
		dir := t.TempDir()
		writeFileAt(t, filepath.Join(dir, "app.txt"), "text", time.Now())

		_, err := newestFileWithPattern(dir, "*.log")
		if err == nil {
			t.Fatal("newestFileWithPattern error = nil")
		}
		requireContains(t, err.Error(), "no files matching pattern")
	})

	t.Run("invalid glob pattern", func(t *testing.T) {
		dir := t.TempDir()
		writeFileAt(t, filepath.Join(dir, "app.log"), "log", time.Now())

		_, err := newestFileWithPattern(dir, "[")
		if err == nil {
			t.Fatal("newestFileWithPattern error = nil")
		}
		requireContains(t, err.Error(), "invalid pattern")
	})

	t.Run("missing directory", func(t *testing.T) {
		_, err := newestFileWithPattern(filepath.Join(t.TempDir(), "missing"), "*")
		if err == nil {
			t.Fatal("newestFileWithPattern error = nil")
		}
	})
}

func TestRepeatedStrings(t *testing.T) {
	var nilRepeated *repeatedStrings
	if got := nilRepeated.String(); got != "" {
		t.Fatalf("nil repeatedStrings.String = %q, want empty", got)
	}

	var r repeatedStrings
	for _, value := range []string{" red:ERROR ", "", " green:DEBUG "} {
		if err := r.Set(value); err != nil {
			t.Fatal(err)
		}
	}

	if got, want := len(r), 2; got != want {
		t.Fatalf("len(repeatedStrings) = %d, want %d", got, want)
	}
	if got, want := r.String(), "red:ERROR,green:DEBUG"; got != want {
		t.Fatalf("repeatedStrings.String = %q, want %q", got, want)
	}
}

func TestParseGlobalArgs(t *testing.T) {
	t.Run("global options before command", func(t *testing.T) {
		withReset(t)

		opts, command, args := parseGlobalArgs([]string{
			"--no-logo",
			"--no-color-logo",
			"--color=always",
			"dir",
			"-pattern",
			"*.log",
			"logs",
		})

		if opts.showLogo || opts.colorLogo {
			t.Fatalf("opts = %+v, want both logo options false", opts)
		}
		if command != "dir" {
			t.Fatalf("command = %q, want dir", command)
		}
		if got, want := strings.Join(args, "\x00"), strings.Join([]string{"-pattern", "*.log", "logs"}, "\x00"); got != want {
			t.Fatalf("args = %#v, want %#v", args, []string{"-pattern", "*.log", "logs"})
		}
		if selectedColorMode != colorAlways {
			t.Fatalf("selectedColorMode = %q, want %q", selectedColorMode, colorAlways)
		}
	})

	t.Run("space separated color mode", func(t *testing.T) {
		withReset(t)

		opts, command, args := parseGlobalArgs([]string{"--color", "never", "file", "app.log"})

		if !opts.showLogo || !opts.colorLogo {
			t.Fatalf("opts = %+v, want default logo options true", opts)
		}
		if command != "file" {
			t.Fatalf("command = %q, want file", command)
		}
		if got, want := strings.Join(args, "\x00"), "app.log"; got != want {
			t.Fatalf("args = %#v, want [app.log]", args)
		}
		if selectedColorMode != colorNever {
			t.Fatalf("selectedColorMode = %q, want %q", selectedColorMode, colorNever)
		}
	})

	t.Run("empty args return no command", func(t *testing.T) {
		withReset(t)

		opts, command, args := parseGlobalArgs(nil)

		if !opts.showLogo || !opts.colorLogo {
			t.Fatalf("opts = %+v, want default logo options true", opts)
		}
		if command != "" {
			t.Fatalf("command = %q, want empty", command)
		}
		if args != nil {
			t.Fatalf("args = %#v, want nil", args)
		}
	})
}

func TestShowLogo(t *testing.T) {
	t.Run("simple logo contains version without ANSI", func(t *testing.T) {
		withReset(t)

		var buf bytes.Buffer
		showLogo(&buf, false)

		got := buf.String()
		requireContains(t, got, "Version "+version)
		requireContains(t, got, "Tail with log-rotate follow")
		requireNotContains(t, got, "\x1b[")
	})

	t.Run("colored logo emits ANSI when forced", func(t *testing.T) {
		withReset(t)
		setColorMode("always")

		var buf bytes.Buffer
		showLogo(&buf, true)

		got := buf.String()
		requireContains(t, got, "Version "+version)
		requireContains(t, got, "\x1b[")
		if plain := stripANSI(got); !strings.Contains(plain, "Tail with log-rotate follow") {
			t.Fatalf("plain colored logo = %q, want tagline", plain)
		}
	})
}

type helperResult struct {
	stdout string
	stderr string
	code   int
}

func TestTrailHelperProcess(t *testing.T) {
	if os.Getenv("TRAIL_TEST_HELPER") != "1" {
		return
	}

	resetTestState()
	log.SetFlags(0)

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("TRAIL_TEST_ARGS")), &args); err != nil {
		log.Fatalf("failed to decode TRAIL_TEST_ARGS: %v", err)
	}

	os.Args = append([]string{"trail"}, args...)
	main()
	os.Exit(0)
}

func runTrailHelper(t *testing.T, args ...string) helperResult {
	t.Helper()

	encodedArgs, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestTrailHelperProcess$")
	cmd.Env = append(os.Environ(),
		"TRAIL_TEST_HELPER=1",
		"TRAIL_TEST_ARGS="+string(encodedArgs),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	result := helperResult{}
	if err := cmd.Run(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		result.code = exitErr.ExitCode()
	}
	result.stdout = stdout.String()
	result.stderr = stderr.String()
	return result
}

func TestMainExitBehaviors(t *testing.T) {
	t.Run("version", func(t *testing.T) {
		result := runTrailHelper(t, "--version")

		if result.code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr=%q", result.code, result.stderr)
		}
		if got, want := result.stdout, version+"\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
		if result.stderr != "" {
			t.Fatalf("stderr = %q, want empty", result.stderr)
		}
	})

	t.Run("help without logo", func(t *testing.T) {
		result := runTrailHelper(t, "--no-logo", "--help")

		if result.code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr=%q", result.code, result.stderr)
		}
		requireContains(t, result.stdout, "USAGE")
		requireContains(t, result.stdout, "COMMON OPTIONS")
		requireNotContains(t, result.stdout, "████")
		if result.stderr != "" {
			t.Fatalf("stderr = %q, want empty", result.stderr)
		}
	})

	t.Run("missing command writes usage to stderr", func(t *testing.T) {
		result := runTrailHelper(t, "--no-logo")

		if result.code != 1 {
			t.Fatalf("exit code = %d, want 1", result.code)
		}
		if result.stdout != "" {
			t.Fatalf("stdout = %q, want empty", result.stdout)
		}
		requireContains(t, result.stderr, "USAGE")
		requireNotContains(t, result.stderr, "████")
	})

	t.Run("invalid color mode fails", func(t *testing.T) {
		result := runTrailHelper(t, "--color", "invalid")

		if result.code != 1 {
			t.Fatalf("exit code = %d, want 1", result.code)
		}
		if result.stdout != "" {
			t.Fatalf("stdout = %q, want empty", result.stdout)
		}
		requireContains(t, result.stderr, `invalid --color value "invalid"`)
	})
}
