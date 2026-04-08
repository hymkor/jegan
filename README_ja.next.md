Jegan - ターミナル用JSONエディター
==================================
<!-- go run github.com/hymkor/example-into-readme/cmd/badges@latest | -->
[![License](https://img.shields.io/badge/License-MIT-red)](https://github.com/hymkor/jegan/blob/master/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/hymkor/jegan.svg)](https://pkg.go.dev/github.com/hymkor/jegan)
[![GitHub](https://img.shields.io/badge/github-repo-blue?logo=github)](https://github.com/hymkor/jegan)
<!-- -->
( [English](README.md) / Japanese )

![](./demo.gif)

特徴
----

- ターミナルのユーザインターフェイスで JSON/JSONL を編集
- ロード時の書式を維持
  - 項目周辺の空白
  - インデントと改行
  - オブジェクトのキーの順番
  - リテラルの表現（エスケープシーケンスなど）
  - ファイル末尾の JSON として解釈できないデータ
- 保存する際の不要な変更を最小化
- ユーザにより変更された項目を強調表示
- JSON はファイルもしくは標準入力から読み取り可能
- vi 風のカーソル移動(`j`/`k`)、編集コマンド(`r`,`o`,`d`)
- 項目編集に Emacs風 readline をサポート

インストール
-----------

### Manual Installation

[Releases](https://github.com/hymkor/jegan/releases) よりバイナリパッケージをダウンロードして、実行ファイルを展開してください

> &#9888;&#65039; Note: macOS用バイナリは実験的ビルドで、検証できていません。
> もし何らかの問題を確認されましたらお知らせください！

<!-- go run github.com/hymkor/example-into-readme/cmd/how2install@latest ja | -->

### [eget] インストーラーを使う場合 (クロスプラットフォーム)

```sh
brew install eget        # Unix-like systems
# or
scoop install eget       # Windows

cd (YOUR-BIN-DIRECTORY)
eget hymkor/jegan
```

[eget]: https://github.com/zyedidia/eget

### [scoop] インストーラーを使う場合 (Windowsのみ)

```
scoop install https://raw.githubusercontent.com/hymkor/jegan/master/jegan.json
```

もしくは

```
scoop bucket add hymkor https://github.com/hymkor/scoop-bucket
scoop install jegan
```

[scoop]: https://scoop.sh/

### "go install" を使う場合 (要Go言語開発環境)

```
go install github.com/hymkor/jegan@latest
```

`go install` は `$HOME/go/bin` もしくは `$GOPATH/bin` へ実行ファイルを導入するので、`jegan` を実行するにはそのディレクトリを `$PATH` に追加する必要があります。
<!-- -->

起動方法
--------

```
jegan some.json
```

もしくは

```
jegan < some.json
```

キー操作
--------

- `j`, `↓`, `Ctrl-N` : 次の項目へ移動
- `k`, `↑`, `Ctrl-P` : 前の項目へ移動
- `l`, `→`, `Ctrl-F` : 表示範囲を右にスクロール
- `h`, `←`, `Ctrl-B` : 表示範囲を左にスクロール
- `<` : 最初の項目へ移動
- `>` : 最後の項目へ移動
- `o` : カーソル行の下へ項目を追加。
  - オブジェクトの項目の場合はキーと値を入力する
  - 配列の項目の場合は値のみを入力する
  - キーは入力された文字列をそのまま使用する（二重引用符不要）
  - 値の型は入力内容に応じて次のように解釈する
    - `"..."` → 文字列（エスケープ文字を解釈）
    - 数値として解釈できるもの → 数値
    - `null` → null
    - `true` / `false` → 真偽値
    - `{}` → 空のオブジェクト
    - `[]` → 空の配列
    - 上記以外 → 文字列（そのまま解釈）
  - Ctrl+G 押下で項目追加をキャンセルできます。
  - 空入力は空文字列（`""`）として扱われる
  - オブジェクトのキーは重複できない
- `r` : カーソル行の項目を変更する（入力方法は `o` と同じ）
- `R` : カーソル行の項目を変更する（値の型を明示的に指定する）
- `d` : カーソル行の項目を削除する。
  ただし、空ではないオブジェクト・配列は削除できない
- `w` : ファイルへ保存
- `q` : 終了

環境変数
--------

### RUNEWIDTH\_EASTASIAN

Unicode で「曖昧幅」とされる文字の表示桁数を明示的に指定します。

- 2桁幅にする場合：`set RUNEWIDTH_EASTASIAN=1`
- 1桁幅にする場合：`set RUNEWIDTH_EASTASIAN=0`（`1` 以外の任意の1文字以上で可）

### GOREADLINESKK

環境変数 `GOREADLINESKK` に辞書ファイルを指定すると、[go-readline-skk] を利用した内蔵 SKK かな漢字変換[^SKK]が有効になります。

- **Windows**
  - `set GOREADLINESKK=SYSTEMJISYOPATH1;SYSTEMJISYOPATH2...;user=USERJISYOPATH`
  - 例:
    `set GOREADLINESKK=~/Share/Etc/SKK-JISYO.L;~/Share/Etc/SKK-JISYO.emoji;user=~/.go-skk-jisyo`
- **Linux**
  - `export GOREADLINESKK=SYSTEMJISYOPATH1:SYSTEMJISYOPATH2...:user=USERJISYOPATH`

（注）`~` は Windows の `cmd.exe` 上でもアプリ側で `%USERPROFILE%` に自動展開されます。

[^SKK]: Simple Kana to Kanji conversion program. One of the Japanese input method editors.

[go-readline-skk]: https://github.com/nyaosorg/go-readline-skk

Changelog
---------

- [English](CHANGELOG.md)
- [Japanese](CHANGELOG_ja.md)

Author
------

- [HAYAMA Kaoru](https://github.com/hymkor)
