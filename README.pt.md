goja
====

Implementação ECMAScript 5.1(+) em Go.

[![Go Reference](https://pkg.go.dev/badge/github.com/dop251/goja.svg)](https://pkg.go.dev/github.com/dop251/goja)

> **Nota sobre este fork**: Este fork inclui suporte extensivo de depuração para o Goja. Tentamos contribuir com essas mudanças para o projeto original através de um pull request mínimo, mas devido ao escopo e momento das mudanças, não foi aceito naquele momento. Como o suporte de depuração é crítico para nossos projetos e para melhorar a qualidade das integrações, continuamos o desenvolvimento neste fork. Apreciamos profundamente o excelente trabalho de dop251 e de todo o projeto Goja. Entendemos que integrar recursos substanciais de depuração requer consideração cuidadosa, e respeitamos a decisão do mantenedor. Este fork permanece disponível para aqueles que precisam de capacidades de depuração agora. Se dop251 decidir fundir essas mudanças com o projeto upstream no futuro, nos traria grande alegria e satisfação ver nossas contribuições beneficiando a comunidade mais ampla do Goja.
>
> **Usando este fork**: Para usar este fork em seu projeto Go, adicione a seguinte diretiva replace ao seu arquivo `go.mod`:
> ```
> replace github.com/dop251/goja => github.com/arturoeanton/goja latest
> ```

Goja é uma implementação de ECMAScript 5.1 em Go puro com ênfase em conformidade com padrões e
desempenho.

Este projeto foi amplamente inspirado por [otto](https://github.com/robertkrimen/otto).

A versão mínima necessária do Go é 1.20.

Recursos
--------

 * Suporte completo ao ECMAScript 5.1 (incluindo regex e modo estrito).
 * Passa quase todos os [testes tc39](https://github.com/tc39/test262) para os recursos implementados até agora. O objetivo é
   passar em todos. Veja .tc39_test262_checkout.sh para o último commit id funcional.
 * Capaz de executar Babel, compilador TypeScript e praticamente qualquer coisa escrita em ES5.
 * Sourcemaps.
 * A maioria da funcionalidade ES6, ainda em progresso, veja https://github.com/dop251/goja/milestone/1?closed=1

Incompatibilidades conhecidas e ressalvas
-----------------------------------------

### WeakMap
WeakMap é implementado incorporando referências aos valores nas chaves. Isso significa que enquanto
a chave for alcançável, todos os valores associados a ela em qualquer weak map também permanecem alcançáveis e, portanto,
não podem ser coletados pelo garbage collector, mesmo que não sejam referenciados de outra forma, mesmo após o WeakMap ter desaparecido.
A referência ao valor é descartada quando a chave é explicitamente removida do WeakMap ou quando a
chave se torna inalcançável.

Para ilustrar:

```javascript
var m = new WeakMap();
var key = {};
var value = {/* um objeto muito grande */};
m.set(key, value);
value = undefined;
m = undefined; // O valor NÃO se torna coletável pelo garbage collector neste ponto
key = undefined; // Agora sim
// m.delete(key); // Isso também funcionaria
```

A razão para isso é a limitação do runtime do Go. No momento da escrita (versão 1.15), ter um finalizador
definido em um objeto que faz parte de um ciclo de referência torna todo o ciclo não coletável pelo garbage collector. A solução
acima é a única maneira razoável que posso pensar sem envolver finalizadores. Esta é a terceira tentativa
(veja https://github.com/dop251/goja/issues/250 e https://github.com/dop251/goja/issues/199 para mais detalhes).

Note que isso não tem nenhum efeito na lógica da aplicação, mas pode causar um uso de memória maior do que o esperado.

### WeakRef e FinalizationRegistry
Pela razão mencionada acima, implementar WeakRef e FinalizationRegistry não parece ser possível neste estágio.

### JSON
`JSON.parse()` usa a biblioteca padrão Go que opera em UTF-8. Portanto, não pode analisar corretamente pares substitutos UTF-16 quebrados, por exemplo:

```javascript
JSON.parse(`"\\uD800"`).charCodeAt(0).toString(16) // retorna "fffd" em vez de "d800"
```

### Date
A conversão de data do calendário para timestamp epoch usa a biblioteca padrão Go que usa `int`, em vez de `float` conforme a
especificação ECMAScript. Isso significa que se você passar argumentos que estouram int para o construtor `Date()` ou se houver
um estouro de inteiro, o resultado será incorreto, por exemplo:

```javascript
Date.UTC(1970, 0, 1, 80063993375, 29, 1, -288230376151711740) // retorna 29256 em vez de 29312
```

FAQ
---

### Quão rápido é?

Embora seja mais rápido que muitas implementações de linguagens de script em Go que vi
(por exemplo, é 6-7 vezes mais rápido que otto em média), não é um
substituto para V8 ou SpiderMonkey ou qualquer outro motor JavaScript de propósito geral.
Você pode encontrar alguns benchmarks [aqui](https://github.com/dop251/goja/issues/2).

### Por que eu usaria isso em vez de um wrapper V8?

Depende muito do seu cenário de uso. Se a maior parte do trabalho é feita em javascript
(por exemplo, criptografia ou outros cálculos pesados), você definitivamente está melhor com V8.

Se você precisa de uma linguagem de script que dirija um motor escrito em Go, de modo que
você precise fazer chamadas frequentes entre Go e javascript passando estruturas de dados complexas,
então o overhead do cgo pode superar os benefícios de ter um motor javascript mais rápido.

Por ser escrito em Go puro, não há dependências cgo, é muito fácil de construir e
deve funcionar em qualquer plataforma suportada pelo Go.

Ele lhe dá muito melhor controle sobre o ambiente de execução, então pode ser útil para pesquisa.

### É seguro para goroutines?

Não. Uma instância de goja.Runtime só pode ser usada por uma única goroutine
por vez. Você pode criar quantas instâncias de Runtime quiser, mas
não é possível passar valores de objetos entre runtimes.

### Onde está setTimeout()/setInterval()?

setTimeout() e setInterval() são funções comuns para fornecer execução concorrente em ambientes ECMAScript, mas as duas funções não fazem parte do padrão ECMAScript.
Navegadores e NodeJS apenas fornecem funções similares, mas não idênticas. A aplicação host precisa controlar o ambiente para execução concorrente, por exemplo, um loop de eventos, e fornecer a funcionalidade ao código script.

Há um [projeto separado](https://github.com/dop251/goja_nodejs) destinado a fornecer alguma funcionalidade NodeJS,
e inclui um loop de eventos.

### Você pode implementar (recurso X do ES6 ou superior)?

Estarei adicionando recursos em sua ordem de dependência e tão rapidamente quanto o tempo permitir. Por favor, não pergunte
por ETAs. Recursos que estão abertos no [milestone](https://github.com/dop251/goja/milestone/1) estão em progresso
ou serão trabalhados em seguida.

O trabalho em andamento é feito em branches de recursos separados que são mesclados no master quando apropriado.
Cada commit nesses branches representa um estado relativamente estável (ou seja, compila e passa em todos os testes tc39 habilitados),
no entanto, como a versão dos testes tc39 que uso é bastante antiga, pode não ser tão bem testada quanto a funcionalidade ES5.1. Como geralmente não há grandes mudanças que quebram a compatibilidade entre as revisões do ECMAScript,
não deve quebrar seu código existente. Você é encorajado a experimentar e relatar quaisquer bugs encontrados. Por favor, não envie correções sem discutir primeiro, pois o código pode mudar nesse meio tempo.

### Como posso contribuir?

Antes de enviar um pull request, certifique-se de que:

- Você seguiu o padrão ECMA o mais próximo possível. Se estiver adicionando um novo recurso, certifique-se de ter lido a especificação,
não baseie apenas em alguns exemplos que funcionam bem.
- Sua mudança não tem um impacto negativo significativo no desempenho (a menos que seja uma correção de bug e seja inevitável)
- Passa em todos os testes tc39 relevantes.

Status Atual
------------

 * Não deve haver mudanças que quebrem a compatibilidade na API, no entanto, ela pode ser estendida.
 * Falta alguma funcionalidade AnnexB.

Exemplo Básico
--------------

Execute JavaScript e obtenha o valor do resultado.

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

Passando Valores para JS
------------------------
Qualquer valor Go pode ser passado para JS usando o método Runtime.ToValue(). Veja a [documentação](https://pkg.go.dev/github.com/dop251/goja#Runtime.ToValue) do método para mais detalhes.

Exportando Valores do JS
------------------------
Um valor JS pode ser exportado para sua representação Go padrão usando o método Value.Export().

Alternativamente, pode ser exportado para uma variável Go específica usando o método [Runtime.ExportTo()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ExportTo).

Dentro de uma única operação de exportação, o mesmo Object será representado pelo mesmo valor Go (seja o mesmo map, slice ou
um ponteiro para a mesma struct). Isso inclui objetos circulares e torna possível exportá-los.

Chamando funções JS do Go
-------------------------
Existem 2 abordagens:

- Usando [AssertFunction()](https://pkg.go.dev/github.com/dop251/goja#AssertFunction):
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
- Usando [Runtime.ExportTo()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ExportTo):
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

fmt.Println(sum(40, 2)) // nota, o valor _this_ na função será undefined.
// Output: 42
```

O primeiro é mais de baixo nível e permite especificar o valor _this_, enquanto o segundo faz a função parecer
uma função Go normal.

Mapeando nomes de campos e métodos de struct
---------------------------------------------
Por padrão, os nomes são passados como estão, o que significa que estão em maiúsculas. Isso não corresponde
à convenção de nomenclatura JavaScript padrão, então se você precisar fazer seu código JS parecer mais natural ou se estiver
lidando com uma biblioteca de terceiros, você pode usar um [FieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#FieldNameMapper):

```go
vm := goja.New()
vm.SetFieldNameMapper(TagFieldNameMapper("json", true))
type S struct {
    Field int `json:"field"`
}
vm.Set("s", S{Field: 42})
res, _ := vm.RunString(`s.field`) // sem o mapper seria s.Field
fmt.Println(res.Export())
// Output: 42
```

Existem dois mappers padrão: [TagFieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#TagFieldNameMapper) e
[UncapFieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#UncapFieldNameMapper), ou você pode usar sua própria implementação.

Construtores Nativos
--------------------

Para implementar uma função construtora em Go, use `func (goja.ConstructorCall) *goja.Object`.
Veja a documentação de [Runtime.ToValue()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ToValue) para mais detalhes.

Expressões Regulares
--------------------

Goja usa a biblioteca regexp incorporada do Go quando possível, caso contrário, recorre ao [regexp2](https://github.com/dlclark/regexp2).

Exceções
--------

Qualquer exceção lançada em JavaScript é retornada como um erro do tipo *Exception. É possível extrair o valor lançado
usando o método Value():

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

Se uma função Go nativa entrar em pânico com um Value, ela é lançada como uma exceção Javascript (e portanto pode ser capturada):

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

Interrupção
-----------

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
    // err é do tipo *InterruptError e seu método Value() retorna o que foi passado para vm.Interrupt()
}
```

Compatibilidade com NodeJS
--------------------------

Há um [projeto separado](https://github.com/dop251/goja_nodejs) destinado a fornecer alguma funcionalidade do NodeJS.