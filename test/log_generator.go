package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	fmt.Println("=== Log Generator for Trail Testing ===")
	
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
	
	// 初期ログファイルを作成
	fmt.Println("Creating initial log file...")
	createInitialLog(testFile)
	
	// ログの継続的な出力
	fmt.Println("Starting continuous log output...")
	fmt.Println("Press Ctrl+C to stop")
	
	go continuousLogOutput(testFile)
	
	// ログローテーションのシミュレーション
	go logRotationSimulation(testDir, testFile)
	
	// メインループ（Ctrl+Cで終了）
	select {}
}

// 初期ログファイルを作成
func createInitialLog(filePath string) {
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

// 継続的なログ出力
func continuousLogOutput(filePath string) {
	counter := 1
	for {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		
		// ランダムなログレベルを選択
		levels := []string{"INFO", "DEBUG", "ERROR", "WARN"}
		level := levels[counter%len(levels)]
		
		var message string
		switch level {
		case "INFO":
			message = fmt.Sprintf("Processing request %d", counter)
		case "DEBUG":
			message = fmt.Sprintf("Memory usage: %dMB", 30+counter%50)
		case "ERROR":
			message = fmt.Sprintf("Error occurred in request %d", counter)
		case "WARN":
			message = fmt.Sprintf("Warning: High load detected for request %d", counter)
		}
		
		logLine := fmt.Sprintf("%s %s %s\n", timestamp, level, message)
		
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Failed to open log file: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		
		if _, err := file.WriteString(logLine); err != nil {
			log.Printf("Failed to append log: %v", err)
		}
		file.Close()
		
		counter++
		time.Sleep(2 * time.Second) // 2秒間隔でログを出力
	}
}

// ログローテーションのシミュレーション
func logRotationSimulation(testDir, testFile string) {
	rotationCounter := 1
	for {
		// 30秒ごとにログローテーションを実行
		time.Sleep(30 * time.Second)
		
		fmt.Printf("\n=== Performing log rotation #%d ===\n", rotationCounter)
		
		// 現在のファイルをリネーム
		rotatedFile := testFile + "." + strconv.FormatInt(time.Now().Unix(), 10)
		if err := os.Rename(testFile, rotatedFile); err != nil {
			log.Printf("Failed to rename file: %v", err)
			continue
		}
		
		fmt.Printf("Rotated log file to: %s\n", filepath.Base(rotatedFile))
		
		// 新しいログファイルを作成
		createInitialLog(testFile)
		fmt.Printf("Created new log file: %s\n", filepath.Base(testFile))
		
		rotationCounter++
	}
} 