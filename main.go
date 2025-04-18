package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
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
	sortOrder := flag.String("sort", "", "並び替え順序: closest (最近のものを優先), reverse (最新から古い順)")
	fromDate := flag.String("from", "", "開始日 (YYYYMMDD形式、例: 20200101)")
	toDate := flag.String("to", "", "終了日 (YYYYMMDD形式、例: 20221231)")
	showLatest := flag.Bool("latest", false, "最新のスナップショットを優先的に取得する (sortパラメータにclosestを設定)")
	yearlySnapshots := flag.Bool("yearly", false, "最古から一定間隔ごとのスナップショットを取得する")
	numYears := flag.Int("num-years", 0, "取得する年数 (yearlyオプションと共に使用。0の場合は現在まで全て取得)")
	interval := flag.Int("interval", 1, "yearly指定時のスナップショット取得間隔（年単位、デフォルト: 1）")
	verbose := flag.Bool("verbose", false, "詳細なログを出力する")
	flag.Parse()

	if *url == "" {
		fmt.Println("エラー: URLを指定してください (-url フラグを使用)")
		fmt.Println("使用例: go run main.go -url example.com")
		os.Exit(1)
	}

	// 最新優先フラグが指定されている場合、sortパラメータを上書き
	if *showLatest && *sortOrder == "" {
		*sortOrder = "closest"
	}

	// URLからプロトコル部分を削除
	cleanURL := *url
	cleanURL = strings.TrimPrefix(cleanURL, "http://")
	cleanURL = strings.TrimPrefix(cleanURL, "https://")
	cleanURL = strings.TrimSuffix(cleanURL, "/")

	// URLに余計なクエリパラメータがある場合は削除
	if idx := strings.Index(cleanURL, "?"); idx > 0 {
		cleanURL = cleanURL[:idx]
	}

	if *verbose {
		fmt.Printf("処理対象URL: %s\n", cleanURL)
	}
	fmt.Printf("%sの履歴を検索中...\n", cleanURL)

	var snapshots []Snapshot
	var err error

	// 1年ごとのスナップショットを取得する場合
	if *yearlySnapshots {
		snapshots, err = getPeriodicSnapshots(*url, *numYears, *interval, *verbose)
		if err != nil {
			fmt.Printf("エラー: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 通常のスナップショット取得
		snapshots, err = getSnapshots(*url, *limit, *sortOrder, *fromDate, *toDate)
		if err != nil {
			fmt.Printf("エラー: %v\n", err)
			os.Exit(1)
		}
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

// getPeriodicSnapshots は最古から指定間隔ごとのスナップショットを取得する
func getPeriodicSnapshots(url string, numYears, yearInterval int, verbose bool) ([]Snapshot, error) {
	if yearInterval <= 0 {
		yearInterval = 1 // デフォルトは1年間隔
	}

	// まず対象URLの全てのスナップショットを取得
	// 件数を大きくして可能な限り多くのスナップショットを取得
	if verbose {
		fmt.Println("全てのスナップショットを取得中...")
	}
	allSnapshots, err := getSnapshots(url, 5000, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("スナップショット取得失敗: %v", err)
	}

	if len(allSnapshots) == 0 {
		return []Snapshot{}, nil
	}

	// スナップショットが1つしかない場合はそれを返す
	if len(allSnapshots) == 1 {
		return allSnapshots, nil
	}

	// 最古のスナップショットを取得
	oldest := allSnapshots[0]
	result := []Snapshot{oldest}

	// タイムスタンプを解析
	oldestTime, err := time.Parse("20060102150405", oldest.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("タイムスタンプ解析失敗: %v", err)
	}

	// 最新のスナップショットを取得
	newestSnapshot := allSnapshots[len(allSnapshots)-1]
	newestTime, err := time.Parse("20060102150405", newestSnapshot.Timestamp)
	if err != nil {
		// エラーが発生した場合は現在の時刻を使用
		newestTime = time.Now()
	}

	// もし検索結果が少なくて最近のスナップショットがない場合は、
	// 逆順でも追加で検索して最新のスナップショットを取得
	if newestTime.Year() < time.Now().Year()-2 {
		if verbose {
			fmt.Println("最新のスナップショットを追加取得中...")
		}
		recentSnapshots, err := getSnapshots(url, 100, "reverse", "", "")
		if err == nil && len(recentSnapshots) > 0 {
			allSnapshots = append(allSnapshots, recentSnapshots...)
			// 最新のスナップショット時刻を更新
			newestSnapshot = recentSnapshots[0]
			newestTime, _ = time.Parse("20060102150405", newestSnapshot.Timestamp)
		} else if verbose && err != nil {
			fmt.Printf("最新スナップショット取得中にエラー発生: %v\n", err)
		}
	}

	fmt.Printf("最古のスナップショット: %s、最新のスナップショット: %s\n",
		oldestTime.Format("2006-01-02"), newestTime.Format("2006-01-02"))
	fmt.Printf("%d年から%d年までの履歴を%d年間隔で検索します（%d件のスナップショットから選択）\n",
		oldestTime.Year(), newestTime.Year(), yearInterval, len(allSnapshots))

	// 最古の日付から指定間隔ずつ進めて、その時点に最も近いスナップショットを取得
	yearCount := 1
	for year := oldestTime.Year() + yearInterval; year <= newestTime.Year(); year += yearInterval {
		// numYearsが指定されている場合、指定された年数に達したら終了
		if numYears > 0 && yearCount >= numYears {
			break
		}

		// この年の日付を生成（最古のスナップショットと同じ月日を使用）
		targetDate := time.Date(year, oldestTime.Month(), oldestTime.Day(), 0, 0, 0, 0, time.UTC)

		if verbose {
			fmt.Printf("%d年のスナップショットを検索中...\n", year)
		}

		// 目標日付に最も近いスナップショットを探す
		var closestSnapshot *Snapshot
		var closestDiff float64 = math.MaxFloat64

		for i := range allSnapshots {
			snapTime, err := time.Parse("20060102150405", allSnapshots[i].Timestamp)
			if err != nil {
				continue
			}

			// 日付の差を計算（絶対値）
			diff := math.Abs(snapTime.Sub(targetDate).Hours() / 24)

			// より近いスナップショットが見つかった場合、更新
			if diff < closestDiff {
				closestDiff = diff
				closestSnapshot = &allSnapshots[i]
			}
		}

		// 最も近いスナップショットが見つかった場合、結果に追加
		if closestSnapshot != nil {
			// 既に追加されたスナップショットと重複しないか確認
			isDuplicate := false
			for _, existingSnap := range result {
				if existingSnap.Timestamp == closestSnapshot.Timestamp {
					isDuplicate = true
					break
				}
			}

			if !isDuplicate {
				result = append(result, *closestSnapshot)
				yearCount++
				if verbose {
					t, _ := time.Parse("20060102150405", closestSnapshot.Timestamp)
					fmt.Printf("  %d年に最も近いスナップショット: %s（差: %.1f日）\n",
						year, t.Format("2006-01-02"), closestDiff)
				}
			} else if verbose {
				fmt.Printf("  %d年のスナップショットは重複のためスキップ\n", year)
			}
		} else if verbose {
			fmt.Printf("  %d年のスナップショットは見つかりませんでした\n", year)
		}
	}

	return result, nil
}

// getSnapshots は指定されたURLのスナップショットを取得する
func getSnapshots(url string, limit int, sortOrder, fromDate, toDate string) ([]Snapshot, error) {
	// URLをエスケープ
	escapedURL := url

	// プロトコルプレフィックスを削除（http://やhttps://）
	escapedURL = strings.TrimPrefix(escapedURL, "http://")
	escapedURL = strings.TrimPrefix(escapedURL, "https://")

	// CDX APIのURL
	apiURL := fmt.Sprintf("http://web.archive.org/cdx/search/cdx?url=%s&output=json&limit=%d", escapedURL, limit)

	// 並び替えオプションを追加
	if sortOrder != "" {
		apiURL += fmt.Sprintf("&sort=%s", sortOrder)
	}

	// 日付範囲を追加
	if fromDate != "" {
		apiURL += fmt.Sprintf("&from=%s", fromDate)
	}
	if toDate != "" {
		apiURL += fmt.Sprintf("&to=%s", toDate)
	}

	// HTTPリクエストを送信
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("APIリクエスト失敗: %v", err)
	}
	defer resp.Body.Close()

	// ステータスコードチェック
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("APIからエラーレスポンス: ステータスコード %d", resp.StatusCode)
	}

	// レスポンスを読み込む
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンス読み込み失敗: %v", err)
	}

	// レスポンスの最初の数バイトをチェックして、JSONかどうか確認
	if len(body) > 0 && body[0] != '[' {
		// デバッグ用に最初の100バイトを出力（もし短ければその全体）
		preview := body
		if len(preview) > 100 {
			preview = preview[:100]
		}
		return nil, fmt.Errorf("JSONではないレスポンスを受信: %s", string(preview))
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
