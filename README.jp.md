goja
====

GoでのECMAScript 5.1(+)実装。

[![Go Reference](https://pkg.go.dev/badge/github.com/dop251/goja.svg)](https://pkg.go.dev/github.com/dop251/goja)

> **このフォークについて**: このフォークには、Gojaの広範なデバッグサポートが含まれています。最小限のプルリクエストを通じてこれらの変更を上流に貢献しようと試みましたが、変更の範囲とタイミングのため、その時点では受け入れられませんでした。デバッグサポートは私たちのプロジェクトにとって重要であり、統合品質の向上に不可欠であるため、このフォークで開発を続けています。dop251とGojaプロジェクト全体の優れた作業を深く評価しています。実質的なデバッグ機能の統合には慎重な検討が必要であることを理解し、メンテナーの決定を尊重します。このフォークは、今すぐデバッグ機能が必要な方々のために利用可能です。dop251が将来これらの変更をアップストリームプロジェクトにマージすることを決定した場合、私たちの貢献がより幅広いGojaコミュニティに役立つことを見ることができ、大きな喜びと満足をもたらすでしょう。
>
> **このフォークの使用方法**: Goプロジェクトでこのフォークを使用するには、`go.mod`ファイルに次のreplaceディレクティブを追加してください：
> ```
> replace github.com/dop251/goja => github.com/arturoeanton/goja latest
> ```

Gojaは、標準準拠とパフォーマンスに重点を置いた純粋なGoでのECMAScript 5.1の実装です。

このプロジェクトは[otto](https://github.com/robertkrimen/otto)に大きく影響を受けています。

必要な最小Goバージョンは1.20です。

機能
----

 * 完全なECMAScript 5.1サポート（正規表現と厳格モードを含む）。
 * これまでに実装された機能について、ほぼすべての[tc39テスト](https://github.com/tc39/test262)に合格。目標は
   すべてに合格すること。最新の動作するコミットIDについては.tc39_test262_checkout.shを参照。
 * Babel、Typescriptコンパイラ、およびES5で書かれたほぼすべてのものを実行可能。
 * ソースマップ。
 * ES6機能の大部分、まだ作業中、https://github.com/dop251/goja/milestone/1?closed=1 を参照

既知の非互換性と注意事項
------------------------

### WeakMap
WeakMapは、キーに値への参照を埋め込むことで実装されています。これは、キーが到達可能である限り、
任意のweak mapでそれに関連付けられたすべての値も到達可能なままであり、したがって
他の方法で参照されていなくても、WeakMapがなくなった後でもガベージコレクションできないことを意味します。
値への参照は、キーがWeakMapから明示的に削除されるか、キーが到達不可能になったときに削除されます。

これを説明するために：

```javascript
var m = new WeakMap();
var key = {};
var value = {/* 非常に大きなオブジェクト */};
m.set(key, value);
value = undefined;
m = undefined; // この時点で値はガベージコレクション可能になりません
key = undefined; // これで可能になります
// m.delete(key); // これも機能します
```

理由はGoランタイムの制限です。執筆時点（バージョン1.15）では、参照サイクルの一部であるオブジェクトにファイナライザーを
設定すると、サイクル全体がガベージコレクション不可能になります。上記の解決策は、
ファイナライザーを含まない唯一の合理的な方法です。これは3回目の試みです
（詳細については https://github.com/dop251/goja/issues/250 と https://github.com/dop251/goja/issues/199 を参照）。

注意：これはアプリケーションロジックに影響を与えませんが、予想以上のメモリ使用量を引き起こす可能性があります。

### WeakRefとFinalizationRegistry
上記の理由により、WeakRefとFinalizationRegistryの実装は現段階では不可能と思われます。

### JSON
`JSON.parse()`はUTF-8で動作する標準のGoライブラリを使用します。したがって、壊れたUTF-16サロゲートペアを正しく解析できません。例：

```javascript
JSON.parse(`"\\uD800"`).charCodeAt(0).toString(16) // "d800"の代わりに"fffd"を返します
```

### Date
カレンダー日付からエポックタイムスタンプへの変換は、ECMAScript仕様の`float`ではなく`int`を使用する標準のGoライブラリを使用します。
これは、`Date()`コンストラクタにintをオーバーフローする引数を渡した場合、または整数オーバーフローがある場合、
結果が正しくないことを意味します。例：

```javascript
Date.UTC(1970, 0, 1, 80063993375, 29, 1, -288230376151711740) // 29312の代わりに29256を返します
```

FAQ
---

### どのくらい速いですか？

私が見た多くのGoでのスクリプト言語実装よりも高速です（例えば、平均してottoより6〜7倍高速）が、
V8やSpiderMonkey、その他の汎用JavaScriptエンジンの代替品ではありません。
いくつかのベンチマークは[こちら](https://github.com/dop251/goja/issues/2)で見つけることができます。

### なぜV8ラッパーよりもこれを使いたいのですか？

使用シナリオに大きく依存します。ほとんどの作業がjavascriptで行われる場合
（例えば、暗号やその他の重い計算）、V8の方が確実に良いでしょう。

Goで書かれたエンジンを駆動するスクリプト言語が必要で、
GoとJavaScript間で複雑なデータ構造を渡して頻繁に呼び出しを行う必要がある場合、
cgoのオーバーヘッドがより高速なJavaScriptエンジンを持つ利点を上回る可能性があります。

純粋なGoで書かれているため、cgoの依存関係がなく、ビルドが非常に簡単で、
Goがサポートするあらゆるプラットフォームで実行できるはずです。

実行環境をより良く制御できるため、研究に役立つ可能性があります。

### goroutineセーフですか？

いいえ。goja.Runtimeのインスタンスは、一度に1つのgoroutineでのみ使用できます。
好きなだけRuntimeのインスタンスを作成できますが、
ランタイム間でオブジェクト値を渡すことはできません。

### setTimeout()/setInterval()はどこですか？

setTimeout()とsetInterval()は、ECMAScript環境で並行実行を提供する一般的な関数ですが、これら2つの関数はECMAScript標準の一部ではありません。
ブラウザとNodeJSは似た機能を提供しますが、同一ではありません。ホスティングアプリケーションは、並行実行の環境（例：イベントループ）を制御し、スクリプトコードに機能を提供する必要があります。

NodeJSの機能の一部を提供することを目的とした[別のプロジェクト](https://github.com/dop251/goja_nodejs)があり、
イベントループが含まれています。

### （ES6以上の機能X）を実装できますか？

依存関係の順序で、時間が許す限り迅速に機能を追加します。ETAについては聞かないでください。
[マイルストーン](https://github.com/dop251/goja/milestone/1)で開いている機能は、進行中か
次に作業される予定です。

進行中の作業は、適切な時期にマスターにマージされる別の機能ブランチで行われます。
これらのブランチの各コミットは比較的安定した状態を表します（つまり、コンパイルされ、有効なすべてのtc39テストに合格します）。
ただし、使用しているtc39テストのバージョンがかなり古いため、ES5.1機能ほど十分にテストされていない可能性があります。ECMAScriptリビジョン間に（通常）大きな破壊的変更がないため、
既存のコードを壊すべきではありません。試してみて、見つかったバグを報告することをお勧めします。ただし、最初に議論せずに修正を提出しないでください。その間にコードが変更される可能性があるためです。

### どのように貢献しますか？

プルリクエストを提出する前に、以下を確認してください：

- ECMAスタンダードにできるだけ忠実に従った。新機能を追加する場合は、仕様を読んだことを確認し、
うまく機能するいくつかの例だけに基づかないでください。
- 変更がパフォーマンスに大きな悪影響を与えない（バグ修正で避けられない場合を除く）
- 関連するすべてのtc39テストに合格する。

現在の状態
----------

 * APIに破壊的変更はないはずですが、拡張される可能性があります。
 * AnnexB機能の一部が欠けています。

基本的な例
----------

JavaScriptを実行し、結果値を取得します。

```go
vm := goja.New()
v, err := vm.RunString("2 + 2")
if err != nil {
    panic(err)
}
if num := v.Export().(int64); num != 4 {
    panic(num)
}
```

JSに値を渡す
------------
任意のGo値は、Runtime.ToValue()メソッドを使用してJSに渡すことができます。詳細については、メソッドの[ドキュメント](https://pkg.go.dev/github.com/dop251/goja#Runtime.ToValue)を参照してください。

JSから値をエクスポート
----------------------
JS値は、Value.Export()メソッドを使用してデフォルトのGo表現にエクスポートできます。

代わりに、[Runtime.ExportTo()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ExportTo)メソッドを使用して特定のGo変数にエクスポートできます。

単一のエクスポート操作内で、同じObjectは同じGo値（同じmap、slice、または
同じstructへのポインタ）で表されます。これには循環オブジェクトが含まれ、それらをエクスポートできます。

GoからJS関数を呼び出す
----------------------
2つのアプローチがあります：

- [AssertFunction()](https://pkg.go.dev/github.com/dop251/goja#AssertFunction)を使用：
```go
const SCRIPT = `
function sum(a, b) {
    return +a + b;
}
`

vm := goja.New()
_, err := vm.RunString(SCRIPT)
if err != nil {
    panic(err)
}
sum, ok := goja.AssertFunction(vm.Get("sum"))
if !ok {
    panic("Not a function")
}

res, err := sum(goja.Undefined(), vm.ToValue(40), vm.ToValue(2))
if err != nil {
    panic(err)
}
fmt.Println(res)
// Output: 42
```
- [Runtime.ExportTo()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ExportTo)を使用：
```go
const SCRIPT = `
function sum(a, b) {
    return +a + b;
}
`

vm := goja.New()
_, err := vm.RunString(SCRIPT)
if err != nil {
    panic(err)
}

var sum func(int, int) int
err = vm.ExportTo(vm.Get("sum"), &sum)
if err != nil {
    panic(err)
}

fmt.Println(sum(40, 2)) // 注：関数内の_this_値はundefinedになります。
// Output: 42
```

最初のものはより低レベルで_this_値を指定できますが、2番目のものは関数を
通常のGo関数のように見せます。

構造体フィールドとメソッド名のマッピング
----------------------------------------
デフォルトでは、名前はそのまま渡されるため、大文字になります。これは
標準的なJavaScriptの命名規則と一致しないため、JSコードをより自然に見せる必要がある場合や、
サードパーティのライブラリを扱っている場合は、[FieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#FieldNameMapper)を使用できます：

```go
vm := goja.New()
vm.SetFieldNameMapper(TagFieldNameMapper("json", true))
type S struct {
    Field int `json:"field"`
}
vm.Set("s", S{Field: 42})
res, _ := vm.RunString(`s.field`) // マッパーなしではs.Fieldになります
fmt.Println(res.Export())
// Output: 42
```

2つの標準マッパーがあります：[TagFieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#TagFieldNameMapper)と
[UncapFieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#UncapFieldNameMapper)、または独自の実装を使用できます。

ネイティブコンストラクタ
------------------------

Goでコンストラクタ関数を実装するには、`func (goja.ConstructorCall) *goja.Object`を使用します。
詳細については、[Runtime.ToValue()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ToValue)のドキュメントを参照してください。

正規表現
--------

Gojaは可能な限り組み込みのGo regexpライブラリを使用し、それ以外の場合は[regexp2](https://github.com/dlclark/regexp2)にフォールバックします。

例外
----

JavaScriptでスローされた例外は、*Exception型のエラーとして返されます。Value()メソッドを使用してスローされた値を抽出できます：

```go
vm := goja.New()
_, err := vm.RunString(`

throw("Test");

`)

if jserr, ok := err.(*Exception); ok {
    if jserr.Value().Export() != "Test" {
        panic("wrong value")
    }
} else {
    panic("wrong type")
}
```

ネイティブGo関数がValueでパニックした場合、JavaScript例外としてスローされます（したがってキャッチできます）：

```go
var vm *Runtime

func Test() {
    panic(vm.ToValue("Error"))
}

vm = goja.New()
vm.Set("Test", Test)
_, err := vm.RunString(`

try {
    Test();
} catch(e) {
    if (e !== "Error") {
        throw e;
    }
}

`)

if err != nil {
    panic(err)
}
```

割り込み
--------

```go
func TestInterrupt(t *testing.T) {
    const SCRIPT = `
    var i = 0;
    for (;;) {
        i++;
    }
    `

    vm := goja.New()
    time.AfterFunc(200 * time.Millisecond, func() {
        vm.Interrupt("halt")
    })

    _, err := vm.RunString(SCRIPT)
    if err == nil {
        t.Fatal("Err is nil")
    }
    // errは*InterruptError型で、そのValue()メソッドはvm.Interrupt()に渡されたものを返します
}
```

NodeJS互換性
------------

NodeJSの機能の一部を提供することを目的とした[別のプロジェクト](https://github.com/dop251/goja_nodejs)があります。