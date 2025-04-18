package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// CDXResponse は CDX API からのレスポンスを表す構造体
type CDXResponse [][]string

// Snapshot はウェブサイトのスナップショットを表す構造体
type Snapshot struct {
	Timestamp string
	URL       string
	MimeType  string
	Status    string
	Digest    string
}

func main() {
	// コマンドライン引数を処理
	url := flag.String("url", "", "対象ウェブサイトのURL (例: example.com)")
	limit := flag.Int("limit", 10, "取得するスナップショットの最大数")
	flag.Parse()

	if *url == "" {
		fmt.Println("エラー: URLを指定してください (-url フラグを使用)")
		fmt.Println("使用例: go run main.go -url example.com")
		os.Exit(1)
	}

	// URLからプロトコル部分を削除
	cleanURL := *url
	cleanURL = strings.TrimPrefix(cleanURL, "http://")
	cleanURL = strings.TrimPrefix(cleanURL, "https://")
	cleanURL = strings.TrimSuffix(cleanURL, "/")

	fmt.Printf("%sの履歴を検索中...\n", cleanURL)

	// CDX APIからスナップショットのリストを取得
	snapshots, err := getSnapshots(cleanURL, *limit)
	if err != nil {
		fmt.Printf("エラー: %v\n", err)
		os.Exit(1)
	}

	if len(snapshots) == 0 {
		fmt.Println("スナップショットが見つかりませんでした")
		return
	}

	// 結果を表示
	fmt.Printf("%d個のスナップショットが見つかりました:\n\n", len(snapshots))

	// ターミナルの幅に合わせて表示形式を変更
	fmt.Printf("%-20s %-10s %-20s\n", "日時", "ステータス", "MIME Type")
	fmt.Println(strings.Repeat("-", 60))

	for _, snap := range snapshots {
		// タイムスタンプをフォーマット
		t, err := time.Parse("20060102150405", snap.Timestamp)
		dateStr := snap.Timestamp
		if err == nil {
			dateStr = t.Format("2006-01-02 15:04:05")
		}

		// ステータスが空の場合は "-" を表示
		status := snap.Status
		if status == "" {
			status = "-"
		}

		// まず基本情報を表示
		fmt.Printf("%-20s %-10s %-20s\n", dateStr, status, snap.MimeType)
		// URLは別行に表示
		fmt.Printf("  %s\n\n", snap.URL)
	}
}

// getSnapshots は指定されたURLのスナップショットを取得する
func getSnapshots(url string, limit int) ([]Snapshot, error) {
	// CDX APIのURL
	apiURL := fmt.Sprintf("http://web.archive.org/cdx/search/cdx?url=%s&output=json&limit=%d", url, limit)

	// HTTPリクエストを送信
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("APIリクエスト失敗: %v", err)
	}
	defer resp.Body.Close()

	// レスポンスを読み込む
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンス読み込み失敗: %v", err)
	}

	// JSONをパース
	var cdxResp CDXResponse
	if err := json.Unmarshal(body, &cdxResp); err != nil {
		return nil, fmt.Errorf("JSONパース失敗: %v", err)
	}

	// CDXレスポンスが空の場合
	if len(cdxResp) <= 1 {
		return []Snapshot{}, nil
	}

	// ヘッダー行をスキップして結果を処理
	snapshots := make([]Snapshot, 0, len(cdxResp)-1)
	for i := 1; i < len(cdxResp); i++ {
		row := cdxResp[i]
		// CDXレスポンスの形式: [urlkey, timestamp, original, mimetype, statuscode, digest, length]
		if len(row) >= 6 {
			timestamp := row[1]
			originalURL := row[2]
			mimeType := row[3]
			statusCode := row[4]
			digest := row[5]

			// アーカイブURLを生成
			archiveURL := fmt.Sprintf("http://web.archive.org/web/%s/%s", timestamp, originalURL)

			snapshots = append(snapshots, Snapshot{
				Timestamp: timestamp,
				URL:       archiveURL,
				MimeType:  mimeType,
				Status:    statusCode,
				Digest:    digest,
			})
		}
	}

	return snapshots, nil
}
