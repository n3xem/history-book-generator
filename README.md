# Internet Archive スナップショット取得ツール

このツールは、[Internet Archive](https://archive.org/)のAPIを使用して、特定のウェブサイトの履歴（スナップショット）一覧を取得するGoプログラムです。

## 機能

- Internet ArchiveのCDX APIを使用してウェブサイトのスナップショット履歴を検索
- スナップショットの日時、HTTPステータス、MIMEタイプなどの情報を表示
- アーカイブされたページへの直接リンクを提供
- 最新のスナップショットを優先的に取得するオプション
- 日付範囲指定による検索の絞り込み

## 使い方

### プログラムのビルド

```bash
go build -o wayback-tool
```

### 実行方法

```bash
./wayback-tool -url example.com
```

または、ビルドせずに直接実行する場合:

```bash
go run main.go -url example.com
```

### オプション

- `-url`: 履歴を検索したいウェブサイトのURL（必須）
- `-limit`: 取得するスナップショットの最大数（デフォルト: 10）
- `-latest`: 最新のスナップショットを優先的に取得する（sortパラメータにclosestを設定）
- `-sort`: 並び替え順序
  - `closest`: 最近のものを優先
  - `reverse`: 最新から古い順
- `-from`: 検索開始日（YYYYMMDD形式、例: 20200101）
- `-to`: 検索終了日（YYYYMMDD形式、例: 20221231）

### 使用例

```bash
# example.comの最新10件のスナップショットを表示
go run main.go -url example.com

# wikipedia.orgの最新50件のスナップショットを表示
go run main.go -url wikipedia.org -limit 50

# 最新のスナップショットを優先して取得
go run main.go -url yahoo.co.jp -latest

# 2020年以降のスナップショットを表示
go run main.go -url github.com -from 20200101

# 2020年から2022年までのスナップショットを逆順（新しい順）で表示
go run main.go -url twitter.com -from 20200101 -to 20221231 -sort reverse
```

## 技術的詳細

このプログラムは、Internet ArchiveのCDX Server APIを使用してウェブページのスナップショット情報を取得しています。取得した情報にはタイムスタンプ、元のURL、MIMEタイプ、HTTPステータスコード、コンテンツのダイジェスト値などが含まれます。

### CDX APIのパラメータ

- `url`: 検索対象のURL
- `output=json`: 出力形式をJSONに指定
- `limit`: 取得するスナップショットの最大数
- `sort`: 並び替え順序
  - デフォルト: 古い順（timestamp昇順）
  - `closest`: 現在の日時に最も近いものを優先
  - `reverse`: 最新から古い順（timestamp降順）
- `from`: 検索開始日（YYYYMMDD形式）
- `to`: 検索終了日（YYYYMMDD形式）

CDX APIの詳細については、[Internet Archive CDX Server API](https://archive.org/developers/tutorial-compare-snapshot-wayback.html)のドキュメントを参照してください。
# history-book-generator
