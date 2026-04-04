Jegan - ターミナル用JSONエディター
==================================
<!-- go run github.com/hymkor/example-into-readme/cmd/badges@latest | -->
[![License](https://img.shields.io/badge/License-MIT-red)](https://github.com/hymkor/jegan/blob/master/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/hymkor/jegan.svg)](https://pkg.go.dev/github.com/hymkor/jegan)
[![GitHub](https://img.shields.io/badge/github-repo-blue?logo=github)](https://github.com/hymkor/jegan)
<!-- -->
( [English](README.md) / Japanese )

![](./demo.gif)

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

- `j`, `↓` : 次の項目へ移動
- `k`, `↑` : 前の項目へ移動
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

JSON の書式についての注意
------------------------

ロードされた JSON のフォーマットは、セーブ時に維持されるもの、されないものがあります。

- 改行コードの種類(LF or CRLF)、インデントに使われる空白文字の数・種類(空白 or タブ)は最初の２行からサンプルをとり、セーブ時に使うようにします。
- EOF 直前の改行の有無をチェックし、セーブ時に再現します。
- オブジェクトの項目の順番は維持されます。
- 記号文字前後の空白文字によるレイアウトは
    - `:` の後は、必ず空白文字ひとつおいてから値が並びます
    - `,` の後は、基本的に改行します（改行が一つも存在しない JSON は除く）
    - `[` や `{` の後は要素が１つ以上あれば改行します（改行が一つも存在しない JSON、要素がゼロ個のオブジェクト・配列は除く）

Changelog
---------

- [English](CHANGELOG.md)
- [Japanese](CHANGELOG_ja.md)

Author
------

- [HAYAMA Kaoru](https://github.com/hymkor)
