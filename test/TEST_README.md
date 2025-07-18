# Trail テストプログラム

このディレクトリには、Trailの機能をテストするためのプログラムが含まれています。

## テストプログラム

### 1. test_trail.go (Go版)

Goで書かれたテストプログラムです。

#### 実行方法

```bash
go run test_trail.go
```

#### テスト内容

1. **色付き表示のテスト**
   - テスト用ログファイルを作成
   - `trail file -c "red:ERROR,green:DEBUG,yellow:WARN"` を実行
   - 新しいログメッセージを追加して色付き表示を確認

2. **ログローテーションのテスト**
   - テスト用ログファイルを作成
   - `trail dir -c "red:ERROR,green:DEBUG"` を実行
   - ファイルをリネームしてログローテーションをシミュレート
   - 新しいファイルの監視を確認

### 2. test_trail.bat (バッチ版)

Windows用のバッチファイルです。

#### 実行方法

```bash
test_trail.bat
```

#### テスト内容

Go版と同じテストを実行します。

## テストの流れ

1. **準備**
   - `test_logs` ディレクトリを作成
   - テスト用ログファイル `app.log` を作成

2. **色付き表示テスト**
   - Trailコマンドを実行
   - ERROR（赤）、DEBUG（緑）、WARN（黄）の色付き表示を確認
   - 新しいログメッセージを追加してリアルタイム表示を確認

3. **ログローテーションテスト**
   - ディレクトリ監視モードでTrailコマンドを実行
   - 現在のログファイルをリネーム（ローテーション）
   - 新しいログファイルを作成
   - 自動的に新しいファイルの監視に切り替わることを確認

4. **クリーンアップ**
   - テスト用ディレクトリとファイルを削除

## 期待される結果

### 色付き表示テスト
- ERRORメッセージが赤色で表示される
- DEBUGメッセージが緑色で表示される
- WARNメッセージが黄色で表示される
- 新しいメッセージがリアルタイムで色付き表示される

### ログローテーションテスト
- 初期ログファイルの内容が表示される
- ファイルがリネームされると、新しいファイルの監視に切り替わる
- ローテーション後の新しいメッセージが色付きで表示される

## 注意事項

- テスト実行前に `trail.exe` がビルドされていることを確認してください
- テスト中は色付き出力が表示されるため、ターミナルの色対応を確認してください
- テストプログラムは自動的にクリーンアップを行いますが、エラーが発生した場合は手動で `test_logs` ディレクトリを削除してください

## トラブルシューティング

### 色が表示されない場合
- ターミナルが色対応しているか確認
- Windows Terminal または PowerShell を使用することを推奨

### ログローテーションが動作しない場合
- ファイルの権限を確認
- ディレクトリの監視権限を確認

### テストプログラムが終了しない場合
- 手動で `trail.exe` プロセスを終了: `taskkill /f /im trail.exe` 