goja
====

Go语言的ECMAScript 5.1(+)实现。

[![Go Reference](https://pkg.go.dev/badge/github.com/dop251/goja.svg)](https://pkg.go.dev/github.com/dop251/goja)

> **关于此分支的说明**：此分支包含了对Goja的广泛调试支持。我们尝试通过最小的拉取请求将这些更改贡献给上游，但由于更改的范围和时机，当时没有被接受。由于调试支持对我们的项目至关重要，并且对于提高集成质量也很重要，我们在此分支中继续开发。我们深深感谢dop251和整个Goja项目的出色工作。我们理解集成大量调试功能需要仔细考虑，我们尊重维护者的决定。此分支仍然可供那些现在需要调试功能的人使用。如果dop251在未来决定将这些更改与上游项目合并，看到我们的贡献惠及更广泛的Goja社区，将会给我们带来极大的喜悦和满足。
>
> **使用此分支**：要在您的Go项目中使用此分支，请在您的`go.mod`文件中添加以下replace指令：
> ```
> replace github.com/dop251/goja => github.com/arturoeanton/goja latest
> ```

Goja是用纯Go实现的ECMAScript 5.1，重点关注标准合规性和性能。

这个项目在很大程度上受到了[otto](https://github.com/robertkrimen/otto)的启发。

最低要求的Go版本是1.20。

特性
----

 * 完整的ECMAScript 5.1支持（包括正则表达式和严格模式）。
 * 几乎通过了目前实现功能的所有[tc39测试](https://github.com/tc39/test262)。目标是
   通过所有测试。请参阅.tc39_test262_checkout.sh以获取最新的工作提交ID。
 * 能够运行Babel、Typescript编译器以及几乎所有用ES5编写的代码。
 * 源码映射。
 * 大部分ES6功能，仍在进行中，请参阅 https://github.com/dop251/goja/milestone/1?closed=1

已知的不兼容性和注意事项
------------------------

### WeakMap
WeakMap通过将对值的引用嵌入到键中来实现。这意味着只要键是可达的，
在任何弱映射中与之关联的所有值也保持可达，因此即使它们没有其他引用，
即使在WeakMap消失后也无法被垃圾回收。
当键从WeakMap中显式删除或键变为不可达时，对值的引用才会被丢弃。

为了说明这一点：

```javascript
var m = new WeakMap();
var key = {};
var value = {/* 一个非常大的对象 */};
m.set(key, value);
value = undefined;
m = undefined; // 此时值不会变为可垃圾回收
key = undefined; // 现在可以了
// m.delete(key); // 这也可以工作
```

原因是Go运行时的限制。在编写时（版本1.15），在作为引用循环一部分的对象上设置终结器
会使整个循环无法垃圾回收。上述解决方案是我能想到的唯一合理的方法，
不涉及终结器。这是第三次尝试
（有关更多详细信息，请参阅 https://github.com/dop251/goja/issues/250 和 https://github.com/dop251/goja/issues/199）。

注意，这对应用程序逻辑没有任何影响，但可能会导致高于预期的内存使用。

### WeakRef和FinalizationRegistry
由于上述原因，在现阶段实现WeakRef和FinalizationRegistry似乎不可能。

### JSON
`JSON.parse()`使用标准Go库，该库以UTF-8操作。因此，它无法正确解析损坏的UTF-16代理对，例如：

```javascript
JSON.parse(`"\\uD800"`).charCodeAt(0).toString(16) // 返回"fffd"而不是"d800"
```

### Date
从日历日期到纪元时间戳的转换使用标准Go库，该库使用`int`，而不是根据ECMAScript规范使用`float`。
这意味着如果您将溢出int的参数传递给`Date()`构造函数，或者如果存在整数溢出，
结果将不正确，例如：

```javascript
Date.UTC(1970, 0, 1, 80063993375, 29, 1, -288230376151711740) // 返回29256而不是29312
```

常见问题
--------

### 它有多快？

虽然它比我见过的许多Go中的脚本语言实现要快
（例如，平均比otto快6-7倍），但它不是
V8或SpiderMonkey或任何其他通用JavaScript引擎的替代品。
您可以在[这里](https://github.com/dop251/goja/issues/2)找到一些基准测试。

### 为什么我要使用它而不是V8包装器？

这在很大程度上取决于您的使用场景。如果大部分工作都在javascript中完成
（例如加密或任何其他繁重的计算），您肯定更适合使用V8。

如果您需要一种脚本语言来驱动用Go编写的引擎，以便
您需要在Go和javascript之间频繁调用并传递复杂的数据结构，
那么cgo开销可能会超过拥有更快javascript引擎的好处。

因为它是用纯Go编写的，所以没有cgo依赖项，非常容易构建，
并且应该在Go支持的任何平台上运行。

它让您对执行环境有更好的控制，因此可能对研究有用。

### 它是goroutine安全的吗？

不是。goja.Runtime的实例一次只能由单个goroutine使用。
您可以创建任意数量的Runtime实例，但
无法在运行时之间传递对象值。

### setTimeout()/setInterval()在哪里？

setTimeout()和setInterval()是在ECMAScript环境中提供并发执行的常见函数，但这两个函数不是ECMAScript标准的一部分。
浏览器和NodeJS只是碰巧提供了类似但不相同的函数。宿主应用程序需要控制并发执行的环境，例如事件循环，并向脚本代码提供功能。

有一个[单独的项目](https://github.com/dop251/goja_nodejs)旨在提供一些NodeJS功能，
它包括一个事件循环。

### 您能实现（ES6或更高版本的功能X）吗？

我将按照依赖顺序尽快添加功能。请不要询问预计时间。
[里程碑](https://github.com/dop251/goja/milestone/1)中开放的功能要么正在进行中，
要么将在下一步进行。

正在进行的工作在单独的功能分支中完成，这些分支在适当的时候合并到master中。
这些分支中的每个提交都代表一个相对稳定的状态（即它可以编译并通过所有启用的tc39测试），
但是由于我使用的tc39测试版本相当旧，它可能不如ES5.1功能测试得那么好。因为ECMAScript修订版之间（通常）没有重大突破性更改，
它不应该破坏您现有的代码。鼓励您尝试并报告发现的任何错误。但请不要在没有先讨论的情况下提交修复，因为代码可能在此期间发生变化。

### 我如何贡献？

在提交拉取请求之前，请确保：

- 您尽可能地遵循了ECMA标准。如果添加新功能，请确保您已阅读规范，
不要仅基于几个运行良好的示例。
- 您的更改不会对性能产生重大负面影响（除非它是错误修复并且不可避免）
- 它通过了所有相关的tc39测试。

当前状态
--------

 * API中不应该有突破性的更改，但是它可能会被扩展。
 * 缺少一些AnnexB功能。

基本示例
--------

运行JavaScript并获取结果值。

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

将值传递给JS
------------
任何Go值都可以使用Runtime.ToValue()方法传递给JS。有关更多详细信息，请参阅该方法的[文档](https://pkg.go.dev/github.com/dop251/goja#Runtime.ToValue)。

从JS导出值
----------
JS值可以使用Value.Export()方法导出到其默认的Go表示形式。

或者，可以使用[Runtime.ExportTo()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ExportTo)方法将其导出到特定的Go变量。

在单个导出操作中，相同的Object将由相同的Go值表示（相同的map、slice或
指向相同struct的指针）。这包括循环对象，使得导出它们成为可能。

从Go调用JS函数
--------------
有2种方法：

- 使用[AssertFunction()](https://pkg.go.dev/github.com/dop251/goja#AssertFunction)：
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
- 使用[Runtime.ExportTo()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ExportTo)：
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

fmt.Println(sum(40, 2)) // 注意，函数中的_this_值将是undefined。
// Output: 42
```

第一种方法更底层，允许指定_this_值，而第二种方法使函数看起来像
普通的Go函数。

映射结构体字段和方法名称
------------------------
默认情况下，名称按原样传递，这意味着它们是大写的。这与
标准JavaScript命名约定不匹配，因此如果您需要使JS代码看起来更自然，或者如果您正在
处理第三方库，您可以使用[FieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#FieldNameMapper)：

```go
vm := goja.New()
vm.SetFieldNameMapper(TagFieldNameMapper("json", true))
type S struct {
    Field int `json:"field"`
}
vm.Set("s", S{Field: 42})
res, _ := vm.RunString(`s.field`) // 没有映射器的话将是s.Field
fmt.Println(res.Export())
// Output: 42
```

有两个标准映射器：[TagFieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#TagFieldNameMapper)和
[UncapFieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#UncapFieldNameMapper)，或者您可以使用自己的实现。

原生构造函数
------------

为了在Go中实现构造函数，请使用`func (goja.ConstructorCall) *goja.Object`。
有关更多详细信息，请参阅[Runtime.ToValue()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ToValue)文档。

正则表达式
----------

Goja在可能的情况下使用嵌入的Go regexp库，否则它会回退到[regexp2](https://github.com/dlclark/regexp2)。

异常
----

JavaScript中抛出的任何异常都作为*Exception类型的错误返回。可以使用Value()方法提取抛出的值：

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

如果原生Go函数使用Value进行panic，它将作为Javascript异常抛出（因此可以被捕获）：

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

中断
----

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
    // err的类型是*InterruptError，其Value()方法返回传递给vm.Interrupt()的任何内容
}
```

NodeJS兼容性
------------

有一个[单独的项目](https://github.com/dop251/goja_nodejs)旨在提供一些NodeJS功能。