# Internet Archive スナップショット取得ツール

このツールは、[Internet Archive](https://archive.org/)のAPIを使用して、特定のウェブサイトの履歴（スナップショット）一覧を取得するGoプログラムです。

## 機能

- Internet ArchiveのCDX APIを使用してウェブサイトのスナップショット履歴を検索
- スナップショットの日時、HTTPステータス、MIMEタイプなどの情報を表示
- アーカイブされたページへの直接リンクを提供

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

### 使用例

```bash
# example.comの最新10件のスナップショットを表示
go run main.go -url example.com

# wikipedia.orgの最新50件のスナップショットを表示
go run main.go -url wikipedia.org -limit 50
```

## 技術的詳細

このプログラムは、Internet ArchiveのCDX Server APIを使用してウェブページのスナップショット情報を取得しています。取得した情報にはタイムスタンプ、元のURL、MIMEタイプ、HTTPステータスコード、コンテンツのダイジェスト値などが含まれます。

CDX APIの詳細については、[Internet Archive CDX Server API](https://archive.org/developers/tutorial-compare-snapshot-wayback.html)のドキュメントを参照してください。 
# history-book-generator
