package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	fmt.Println("=== Trail テストプログラム ===")
	
	// テスト用ディレクトリとファイルの準備
	testDir := "./test_logs"
	testFile := filepath.Join(testDir, "app.log")
	
	// テストディレクトリを作成
	if err := os.MkdirAll(testDir, 0755); err != nil {
		log.Fatalf("テストディレクトリの作成に失敗: %v", err)
	}
	
	// テスト終了時のクリーンアップ
	defer func() {
		fmt.Println("\n=== クリーンアップ ===")
		if err := os.RemoveAll(testDir); err != nil {
			fmt.Printf("クリーンアップエラー: %v\n", err)
		} else {
			fmt.Println("テストファイルを削除しました")
		}
	}()
	
	// テスト1: 色付き表示のテスト
	fmt.Println("\n1. 色付き表示のテスト")
	testColorOutput(testFile)
	
	// テスト2: ログローテーションのテスト
	fmt.Println("\n2. ログローテーションのテスト")
	testLogRotation(testDir, testFile)
	
	fmt.Println("\n=== テスト完了 ===")
}

// 色付き表示のテスト
func testColorOutput(testFile string) {
	fmt.Println("色付き表示をテスト中...")
	
	// テスト用ログファイルを作成
	createTestLog(testFile)
	
	// trailコマンドを実行（非同期）
	cmd := exec.Command("./trail.exe", "file", "-c", "red:ERROR,green:DEBUG,yellow:WARN", testFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Start(); err != nil {
		log.Printf("trailコマンドの開始に失敗: %v", err)
		return
	}
	
	// 少し待ってから新しいログを追加
	time.Sleep(2 * time.Second)
	appendTestLog(testFile, "新しいERRORメッセージ")
	appendTestLog(testFile, "新しいDEBUGメッセージ")
	appendTestLog(testFile, "新しいWARNメッセージ")
	
	// さらに待ってから終了
	time.Sleep(3 * time.Second)
	cmd.Process.Kill()
	
	fmt.Println("色付き表示テスト完了")
}

// ログローテーションのテスト
func testLogRotation(testDir, testFile string) {
	fmt.Println("ログローテーションをテスト中...")
	
	// 初期ログファイルを作成
	createTestLog(testFile)
	
	// trailコマンドを実行（非同期）
	cmd := exec.Command("./trail.exe", "dir", "-c", "red:ERROR,green:DEBUG", testDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Start(); err != nil {
		log.Printf("trailコマンドの開始に失敗: %v", err)
		return
	}
	
	// 少し待ってからログローテーションをシミュレート
	time.Sleep(2 * time.Second)
	
	// ログローテーション: 現在のファイルをリネームして新しいファイルを作成
	rotatedFile := testFile + "." + strconv.FormatInt(time.Now().Unix(), 10)
	if err := os.Rename(testFile, rotatedFile); err != nil {
		log.Printf("ファイルのリネームに失敗: %v", err)
		return
	}
	
	// 新しいログファイルを作成
	createTestLog(testFile)
	appendTestLog(testFile, "ローテーション後のERRORメッセージ")
	appendTestLog(testFile, "ローテーション後のDEBUGメッセージ")
	
	// さらに待ってから終了
	time.Sleep(3 * time.Second)
	cmd.Process.Kill()
	
	fmt.Println("ログローテーションテスト完了")
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
		log.Printf("ログファイルの作成に失敗: %v", err)
	}
}

// ログファイルに新しい行を追加
func appendTestLog(filePath, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("%s %s\n", timestamp, message)
	
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("ログファイルのオープンに失敗: %v", err)
		return
	}
	defer file.Close()
	
	if _, err := file.WriteString(logLine); err != nil {
		log.Printf("ログの追加に失敗: %v", err)
	}
} 