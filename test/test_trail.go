package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

func main() {
	fmt.Println("=== Trail Test Program ===")

	// 実行ファイルの存在確認
	execPath := getExecutablePath()
	if execPath == getExecutableName() {
		fmt.Printf("Warning: Could not find trail executable. Make sure to build the project first:\n")
		fmt.Printf("  go build -o %s\n", getExecutableName())
		fmt.Printf("  or\n")
		fmt.Printf("  go install\n")
		fmt.Println()
	} else {
		fmt.Printf("Found trail executable at: %s\n", execPath)
	}

	// テスト用ディレクトリとファイルの準備
	testDir := "./test_logs"
	testFile := filepath.Join(testDir, "app.log")

	// テストディレクトリを作成
	if err := os.MkdirAll(testDir, 0755); err != nil {
		log.Fatalf("Failed to create test directory: %v", err)
	}

	// テスト終了時のクリーンアップ
	defer func() {
		fmt.Println("\n=== Cleanup ===")
		if err := os.RemoveAll(testDir); err != nil {
			fmt.Printf("Cleanup error: %v\n", err)
		} else {
			fmt.Println("Test files deleted")
		}
	}()

	// テスト1: 色付き表示のテスト
	fmt.Println("\n1. Color Output Test")
	testColorOutput(testFile)

	// テスト2: ログローテーションのテスト
	fmt.Println("\n2. Log Rotation Test")
	testLogRotation(testDir, testFile)

	fmt.Println("\n=== Test Completed ===")
}

// 実行ファイル名を動的に決定
func getExecutableName() string {
	if runtime.GOOS == "windows" {
		return "trail.exe"
	}
	return "trail"
}

// 実行ファイルのパスを取得
func getExecutablePath() string {
	execName := getExecutableName()

	// 親ディレクトリで実行ファイルを探す（testディレクトリから一つ上）
	currentDir, err := os.Getwd()
	if err == nil {
		parentDir := filepath.Dir(currentDir)
		parentPath := filepath.Join(parentDir, execName)
		if _, err := os.Stat(parentPath); err == nil {
			return parentPath
		}

		// 現在のディレクトリでも探す
		currentPath := filepath.Join(currentDir, execName)
		if _, err := os.Stat(currentPath); err == nil {
			return currentPath
		}
	}

	// PATH環境変数から実行ファイルを探す
	if path, err := exec.LookPath(execName); err == nil {
		return path
	}

	// 見つからない場合は実行ファイル名のみを返す
	return execName
}

// 色付き表示のテスト
func testColorOutput(testFile string) {
	fmt.Println("Testing color output...")

	// テスト用ログファイルを作成
	createTestLog(testFile)

	// trailコマンドを実行（非同期）
	execPath := getExecutablePath()
	cmd := exec.Command(execPath, "file", "-c", "red:ERROR,green:DEBUG,yellow:WARN", testFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start trail command: %v", err)
		return
	}

	// 少し待ってから新しいログを追加
	time.Sleep(2 * time.Second)
	appendTestLog(testFile, "New ERROR message")
	appendTestLog(testFile, "New DEBUG message")
	appendTestLog(testFile, "New WARN message")

	// さらに待ってから終了
	time.Sleep(3 * time.Second)
	cmd.Process.Kill()

	fmt.Println("Color output test completed")
}

// ログローテーションのテスト
func testLogRotation(testDir, testFile string) {
	fmt.Println("Testing log rotation...")

	// 初期ログファイルを作成
	createTestLog(testFile)

	// trailコマンドを実行（非同期）
	execPath := getExecutablePath()
	cmd := exec.Command(execPath, "dir", "-c", "red:ERROR,green:DEBUG", testDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start trail command: %v", err)
		return
	}

	// 少し待ってからログローテーションをシミュレート
	time.Sleep(2 * time.Second)

	// ログローテーション: 現在のファイルをリネームして新しいファイルを作成
	rotatedFile := testFile + "." + strconv.FormatInt(time.Now().Unix(), 10)
	if err := os.Rename(testFile, rotatedFile); err != nil {
		log.Printf("Failed to rename file: %v", err)
		return
	}

	// 新しいログファイルを作成
	createTestLog(testFile)
	appendTestLog(testFile, "ERROR message after rotation")
	appendTestLog(testFile, "DEBUG message after rotation")

	// さらに待ってから終了
	time.Sleep(3 * time.Second)
	cmd.Process.Kill()

	fmt.Println("Log rotation test completed")
}

// テスト用ログファイルを作成
func createTestLog(filePath string) {
	content := `2024-01-15 10:30:15 INFO Application started
2024-01-15 10:30:16 DEBUG Loading configuration
2024-01-15 10:30:17 ERROR Failed to connect to database
2024-01-15 10:30:18 WARN Retrying connection...
2024-01-15 10:30:19 INFO Connection established
2024-01-15 10:30:20 DEBUG Processing request 12345
2024-01-15 10:30:21 ERROR Invalid input data
2024-01-15 10:30:22 INFO Request completed
2024-01-15 10:30:23 DEBUG Memory usage: 45MB
2024-01-15 10:30:24 WARN High memory usage detected
2024-01-15 10:30:25 ERROR Out of memory
`

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		log.Printf("Failed to create log file: %v", err)
	}
}

// ログファイルに新しい行を追加
func appendTestLog(filePath, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("%s %s\n", timestamp, message)

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(logLine); err != nil {
		log.Printf("Failed to append log: %v", err)
	}
}
