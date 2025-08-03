goja
====

ECMAScript 5.1(+) Implementierung in Go.

[![Go Reference](https://pkg.go.dev/badge/github.com/dop251/goja.svg)](https://pkg.go.dev/github.com/dop251/goja)

> **Hinweis zu diesem Fork**: Dieser Fork enthält umfangreiche Debugging-Unterstützung für Goja. Wir haben versucht, diese Änderungen über einen minimalen Pull Request an das Upstream-Projekt beizusteuern, aber aufgrund des Umfangs und des Zeitpunkts der Änderungen wurde er zu diesem Zeitpunkt nicht akzeptiert. Da die Debugging-Unterstützung für unsere Projekte kritisch ist und zur Verbesserung der Integrationsqualität wichtig ist, haben wir die Entwicklung in diesem Fork fortgesetzt. Wir schätzen die hervorragende Arbeit von dop251 und dem gesamten Goja-Projekt sehr. Wir verstehen, dass die Integration umfangreicher Debugging-Funktionen sorgfältige Überlegungen erfordert, und respektieren die Entscheidung des Maintainers. Dieser Fork bleibt für diejenigen verfügbar, die jetzt Debugging-Funktionen benötigen. Sollte dop251 sich in Zukunft entscheiden, diese Änderungen mit dem Upstream-Projekt zusammenzuführen, würde es uns große Freude und Befriedigung bereiten, unsere Beiträge der breiteren Goja-Community zugute kommen zu sehen.
>
> **Verwendung dieses Forks**: Um diesen Fork in Ihrem Go-Projekt zu verwenden, fügen Sie die folgende Replace-Anweisung zu Ihrer `go.mod`-Datei hinzu:
> ```
> replace github.com/dop251/goja => github.com/arturoeanton/goja latest
> ```

Goja ist eine Implementierung von ECMAScript 5.1 in reinem Go mit Schwerpunkt auf Standard-Konformität und
Leistung.

Dieses Projekt wurde weitgehend von [otto](https://github.com/robertkrimen/otto) inspiriert.

Die minimal erforderliche Go-Version ist 1.20.

Funktionen
----------

 * Vollständige ECMAScript 5.1 Unterstützung (einschließlich Regex und Strict Mode).
 * Besteht fast alle [tc39 Tests](https://github.com/tc39/test262) für die bisher implementierten Funktionen. Das Ziel ist es,
   alle zu bestehen. Siehe .tc39_test262_checkout.sh für die neueste funktionierende Commit-ID.
 * Kann Babel, Typescript-Compiler und praktisch alles, was in ES5 geschrieben wurde, ausführen.
 * Sourcemaps.
 * Die meisten ES6-Funktionen, noch in Arbeit, siehe https://github.com/dop251/goja/milestone/1?closed=1

Bekannte Inkompatibilitäten und Vorbehalte
-------------------------------------------

### WeakMap
WeakMap wird implementiert, indem Verweise auf die Werte in die Schlüssel eingebettet werden. Das bedeutet, dass solange
der Schlüssel erreichbar ist, alle mit ihm in irgendeiner Weak Map verbundenen Werte ebenfalls erreichbar bleiben und daher
nicht vom Garbage Collector erfasst werden können, selbst wenn sie anderweitig nicht referenziert werden, auch nachdem die WeakMap verschwunden ist.
Der Verweis auf den Wert wird entweder gelöscht, wenn der Schlüssel explizit aus der WeakMap entfernt wird oder wenn der
Schlüssel unerreichbar wird.

Zur Veranschaulichung:

```javascript
var m = new WeakMap();
var key = {};
var value = {/* ein sehr großes Objekt */};
m.set(key, value);
value = undefined;
m = undefined; // Der Wert wird an diesem Punkt NICHT vom Garbage Collector erfassbar
key = undefined; // Jetzt schon
// m.delete(key); // Das würde auch funktionieren
```

Der Grund dafür ist die Einschränkung der Go-Laufzeitumgebung. Zum Zeitpunkt des Schreibens (Version 1.15) macht das Setzen eines Finalizers
auf ein Objekt, das Teil eines Referenzzyklus ist, den gesamten Zyklus nicht vom Garbage Collector erfassbar. Die obige Lösung
ist der einzige vernünftige Weg, den ich mir ohne Finalizer vorstellen kann. Dies ist der dritte Versuch
(siehe https://github.com/dop251/goja/issues/250 und https://github.com/dop251/goja/issues/199 für weitere Details).

Beachten Sie, dass dies keine Auswirkungen auf die Anwendungslogik hat, aber eine höhere als erwartete Speichernutzung verursachen kann.

### WeakRef und FinalizationRegistry
Aus dem oben genannten Grund scheint die Implementierung von WeakRef und FinalizationRegistry in diesem Stadium nicht möglich zu sein.

### JSON
`JSON.parse()` verwendet die Standard-Go-Bibliothek, die in UTF-8 arbeitet. Daher kann es gebrochene UTF-16-Ersatzpaare nicht korrekt parsen, zum Beispiel:

```javascript
JSON.parse(`"\\uD800"`).charCodeAt(0).toString(16) // gibt "fffd" statt "d800" zurück
```

### Date
Die Konvertierung vom Kalenderdatum zum Epochen-Zeitstempel verwendet die Standard-Go-Bibliothek, die `int` anstelle von `float` gemäß
ECMAScript-Spezifikation verwendet. Das bedeutet, wenn Sie Argumente übergeben, die int zum `Date()`-Konstruktor überlaufen lassen, oder wenn es
einen Integer-Überlauf gibt, wird das Ergebnis falsch sein, zum Beispiel:

```javascript
Date.UTC(1970, 0, 1, 80063993375, 29, 1, -288230376151711740) // gibt 29256 statt 29312 zurück
```

FAQ
---

### Wie schnell ist es?

Obwohl es schneller ist als viele Skriptsprachen-Implementierungen in Go, die ich gesehen habe
(zum Beispiel ist es im Durchschnitt 6-7 mal schneller als otto), ist es kein
Ersatz für V8 oder SpiderMonkey oder andere Allzweck-JavaScript-Engines.
Sie können einige Benchmarks [hier](https://github.com/dop251/goja/issues/2) finden.

### Warum sollte ich es anstelle eines V8-Wrappers verwenden?

Es hängt stark von Ihrem Anwendungsfall ab. Wenn der Großteil der Arbeit in JavaScript erledigt wird
(zum Beispiel Krypto oder andere schwere Berechnungen), sind Sie definitiv besser mit V8 dran.

Wenn Sie eine Skriptsprache benötigen, die eine in Go geschriebene Engine antreibt, sodass
Sie häufige Aufrufe zwischen Go und JavaScript mit komplexen Datenstrukturen durchführen müssen,
dann kann der cgo-Overhead die Vorteile einer schnelleren JavaScript-Engine überwiegen.

Da es in reinem Go geschrieben ist, gibt es keine cgo-Abhängigkeiten, es ist sehr einfach zu bauen und
sollte auf jeder von Go unterstützten Plattform laufen.

Es gibt Ihnen eine viel bessere Kontrolle über die Ausführungsumgebung, was für die Forschung nützlich sein kann.

### Ist es goroutine-sicher?

Nein. Eine Instanz von goja.Runtime kann nur von einer einzigen Goroutine
gleichzeitig verwendet werden. Sie können beliebig viele Runtime-Instanzen erstellen, aber
es ist nicht möglich, Objektwerte zwischen Runtimes zu übergeben.

### Wo ist setTimeout()/setInterval()?

setTimeout() und setInterval() sind gängige Funktionen zur Bereitstellung gleichzeitiger Ausführung in ECMAScript-Umgebungen, aber die beiden Funktionen sind nicht Teil des ECMAScript-Standards.
Browser und NodeJS bieten zufällig ähnliche, aber nicht identische Funktionen. Die Host-Anwendung muss die Umgebung für die gleichzeitige Ausführung kontrollieren, z.B. eine Event-Loop, und die Funktionalität dem Skriptcode bereitstellen.

Es gibt ein [separates Projekt](https://github.com/dop251/goja_nodejs), das darauf abzielt, einige NodeJS-Funktionalität bereitzustellen,
und es enthält eine Event-Loop.

### Können Sie (Feature X aus ES6 oder höher) implementieren?

Ich werde Funktionen in ihrer Abhängigkeitsreihenfolge und so schnell wie es die Zeit erlaubt hinzufügen. Bitte fragen Sie nicht
nach ETAs. Funktionen, die im [Meilenstein](https://github.com/dop251/goja/milestone/1) offen sind, sind entweder in Arbeit
oder werden als nächstes bearbeitet.

Die laufende Arbeit wird in separaten Feature-Branches durchgeführt, die bei Bedarf in den Master gemergt werden.
Jeder Commit in diesen Branches repräsentiert einen relativ stabilen Zustand (d.h. er kompiliert und besteht alle aktivierten tc39-Tests),
da jedoch die Version der tc39-Tests, die ich verwende, ziemlich alt ist, ist sie möglicherweise nicht so gut getestet wie die ES5.1-Funktionalität. Da es (normalerweise) keine größeren Breaking Changes zwischen ECMAScript-Revisionen gibt,
sollte es Ihren bestehenden Code nicht brechen. Sie werden ermutigt, es auszuprobieren und gefundene Fehler zu melden. Bitte reichen Sie jedoch keine Fixes ein, ohne es vorher zu besprechen, da sich der Code in der Zwischenzeit ändern könnte.

### Wie kann ich beitragen?

Bevor Sie einen Pull Request einreichen, stellen Sie bitte sicher, dass:

- Sie dem ECMA-Standard so genau wie möglich gefolgt sind. Wenn Sie eine neue Funktion hinzufügen, stellen Sie sicher, dass Sie die Spezifikation gelesen haben,
basieren Sie es nicht nur auf ein paar Beispielen, die gut funktionieren.
- Ihre Änderung hat keine signifikant negative Auswirkung auf die Leistung (es sei denn, es ist ein Bugfix und unvermeidbar)
- Es besteht alle relevanten tc39-Tests.

Aktueller Status
----------------

 * Es sollte keine Breaking Changes in der API geben, sie kann jedoch erweitert werden.
 * Einige der AnnexB-Funktionalität fehlt.

Grundlegendes Beispiel
----------------------

JavaScript ausführen und den Ergebniswert erhalten.

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

Werte an JS übergeben
---------------------
Jeder Go-Wert kann mit der Runtime.ToValue()-Methode an JS übergeben werden. Siehe die [Dokumentation](https://pkg.go.dev/github.com/dop251/goja#Runtime.ToValue) der Methode für weitere Details.

Werte aus JS exportieren
------------------------
Ein JS-Wert kann mit der Value.Export()-Methode in seine Standard-Go-Darstellung exportiert werden.

Alternativ kann er mit der [Runtime.ExportTo()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ExportTo)-Methode in eine spezifische Go-Variable exportiert werden.

Innerhalb einer einzelnen Export-Operation wird dasselbe Object durch denselben Go-Wert dargestellt (entweder dieselbe Map, Slice oder
ein Zeiger auf dieselbe Struct). Dies schließt zirkuläre Objekte ein und macht es möglich, sie zu exportieren.

JS-Funktionen von Go aus aufrufen
----------------------------------
Es gibt 2 Ansätze:

- Mit [AssertFunction()](https://pkg.go.dev/github.com/dop251/goja#AssertFunction):
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
- Mit [Runtime.ExportTo()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ExportTo):
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

fmt.Println(sum(40, 2)) // Hinweis: _this_ Wert in der Funktion wird undefined sein.
// Output: 42
```

Der erste ist niedriger Level und erlaubt die Angabe des _this_ Wertes, während der zweite die Funktion wie
eine normale Go-Funktion aussehen lässt.

Zuordnung von Struct-Feld- und Methodennamen
---------------------------------------------
Standardmäßig werden die Namen unverändert übergeben, was bedeutet, dass sie großgeschrieben sind. Dies entspricht nicht
der Standard-JavaScript-Namenskonvention. Wenn Sie also Ihren JS-Code natürlicher aussehen lassen müssen oder wenn Sie
mit einer Drittanbieter-Bibliothek arbeiten, können Sie einen [FieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#FieldNameMapper) verwenden:

```go
vm := goja.New()
vm.SetFieldNameMapper(TagFieldNameMapper("json", true))
type S struct {
    Field int `json:"field"`
}
vm.Set("s", S{Field: 42})
res, _ := vm.RunString(`s.field`) // ohne den Mapper wäre es s.Field
fmt.Println(res.Export())
// Output: 42
```

Es gibt zwei Standard-Mapper: [TagFieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#TagFieldNameMapper) und
[UncapFieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#UncapFieldNameMapper), oder Sie können Ihre eigene Implementierung verwenden.

Native Konstruktoren
--------------------

Um eine Konstruktor-Funktion in Go zu implementieren, verwenden Sie `func (goja.ConstructorCall) *goja.Object`.
Siehe [Runtime.ToValue()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ToValue) Dokumentation für weitere Details.

Reguläre Ausdrücke
------------------

Goja verwendet die eingebettete Go regexp-Bibliothek wo möglich, andernfalls fällt es auf [regexp2](https://github.com/dlclark/regexp2) zurück.

Ausnahmen
---------

Jede in JavaScript geworfene Ausnahme wird als Fehler vom Typ *Exception zurückgegeben. Es ist möglich, den geworfenen Wert
mit der Value()-Methode zu extrahieren:

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

Wenn eine native Go-Funktion mit einem Value panikt, wird es als Javascript-Ausnahme geworfen (und kann daher gefangen werden):

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

Unterbrechung
-------------

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
    // err ist vom Typ *InterruptError und seine Value()-Methode gibt zurück, was an vm.Interrupt() übergeben wurde
}
```

NodeJS-Kompatibilität
---------------------

Es gibt ein [separates Projekt](https://github.com/dop251/goja_nodejs), das darauf abzielt, einige der NodeJS-Funktionalität bereitzustellen.