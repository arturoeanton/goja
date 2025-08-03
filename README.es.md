goja
====

Implementación de ECMAScript 5.1(+) en Go.

[![Go Reference](https://pkg.go.dev/badge/github.com/dop251/goja.svg)](https://pkg.go.dev/github.com/dop251/goja)

> **Nota sobre este fork**: Este fork incluye soporte extensivo de depuración para Goja. Intentamos contribuir estos cambios al proyecto original a través de un pull request mínimo, pero debido al alcance y momento de los cambios, no fue aceptado en ese momento. Como el soporte de depuración es crítico para nuestros proyectos y para mejorar la calidad de las integraciones, hemos continuado el desarrollo en este fork. Apreciamos profundamente el excelente trabajo de dop251 y todo el proyecto Goja. Entendemos que integrar características sustanciales de depuración requiere una consideración cuidadosa, y respetamos la decisión del mantenedor. Este fork permanece disponible para aquellos que necesiten capacidades de depuración ahora. Si dop251 decidiera fusionar estos cambios con el proyecto principal en el futuro, nos traería gran alegría y satisfacción ver nuestras contribuciones beneficiar a toda la comunidad de Goja.
>
> **Usando este fork**: Para usar este fork en tu proyecto Go, agrega la siguiente directiva replace a tu archivo `go.mod`:
> ```
> replace github.com/dop251/goja => github.com/arturoeanton/goja latest
> ```

Goja es una implementación de ECMAScript 5.1 en Go puro con énfasis en cumplimiento estándar y
rendimiento.

Este proyecto fue inspirado en gran medida por [otto](https://github.com/robertkrimen/otto).

La versión mínima requerida de Go es 1.20.

Características
---------------

 * Soporte completo de ECMAScript 5.1 (incluyendo regex y modo estricto).
 * Pasa casi todas las [pruebas tc39](https://github.com/tc39/test262) para las características implementadas hasta ahora. El objetivo es
   pasar todas ellas. Ver .tc39_test262_checkout.sh para el último commit id funcional.
 * Capaz de ejecutar Babel, compilador de Typescript y prácticamente cualquier cosa escrita en ES5.
 * Sourcemaps.
 * La mayoría de la funcionalidad ES6, aún trabajo en progreso, ver https://github.com/dop251/goja/milestone/1?closed=1

Incompatibilidades conocidas y advertencias
-------------------------------------------

### WeakMap
WeakMap está implementado incrustando referencias a los valores en las claves. Esto significa que mientras
la clave sea alcanzable, todos los valores asociados con ella en cualquier weak map también permanecen alcanzables y por lo tanto
no pueden ser recolectados por el recolector de basura incluso si no están referenciados de otra manera, incluso después de que el WeakMap se haya ido.
La referencia al valor se elimina cuando la clave se elimina explícitamente del WeakMap o cuando la
clave se vuelve inalcanzable.

Para ilustrar esto:

```javascript
var m = new WeakMap();
var key = {};
var value = {/* un objeto muy grande */};
m.set(key, value);
value = undefined;
m = undefined; // El valor NO se vuelve recolectable por basura en este punto
key = undefined; // Ahora sí
// m.delete(key); // Esto también funcionaría
```

La razón es la limitación del runtime de Go. Al momento de escribir (versión 1.15) tener un finalizador
configurado en un objeto que es parte de un ciclo de referencia hace que todo el ciclo no sea recolectable por basura. La solución
anterior es la única forma razonable que puedo pensar sin involucrar finalizadores. Este es el tercer intento
(ver https://github.com/dop251/goja/issues/250 y https://github.com/dop251/goja/issues/199 para más detalles).

Nota, esto no tiene ningún efecto en la lógica de la aplicación, pero puede causar un uso de memoria mayor al esperado.

### WeakRef y FinalizationRegistry
Por la razón mencionada arriba, implementar WeakRef y FinalizationRegistry no parece ser posible en esta etapa.

### JSON
`JSON.parse()` usa la biblioteca estándar de Go que opera en UTF-8. Por lo tanto, no puede analizar correctamente pares surrogados UTF-16 rotos, por ejemplo:

```javascript
JSON.parse(`"\\uD800"`).charCodeAt(0).toString(16) // devuelve "fffd" en lugar de "d800"
```

### Date
La conversión de fecha calendario a timestamp epoch usa la biblioteca estándar de Go que usa `int`, en lugar de `float` según la
especificación ECMAScript. Esto significa que si pasas argumentos que desbordan int al constructor `Date()` o si hay
un desbordamiento de enteros, el resultado será incorrecto, por ejemplo:

```javascript
Date.UTC(1970, 0, 1, 80063993375, 29, 1, -288230376151711740) // devuelve 29256 en lugar de 29312
```

FAQ
---

### ¿Qué tan rápido es?

Aunque es más rápido que muchas implementaciones de lenguajes de scripting en Go que he visto
(por ejemplo, es 6-7 veces más rápido que otto en promedio) no es un
reemplazo para V8 o SpiderMonkey o cualquier otro motor JavaScript de propósito general.
Puedes encontrar algunos benchmarks [aquí](https://github.com/dop251/goja/issues/2).

### ¿Por qué querría usarlo sobre un wrapper de V8?

Depende mucho de tu escenario de uso. Si la mayor parte del trabajo se hace en javascript
(por ejemplo, crypto o cualquier otro cálculo pesado) definitivamente estás mejor con V8.

Si necesitas un lenguaje de scripting que maneje un motor escrito en Go de modo que
necesites hacer llamadas frecuentes entre Go y javascript pasando estructuras de datos complejas
entonces el overhead de cgo puede superar los beneficios de tener un motor javascript más rápido.

Porque está escrito en Go puro no hay dependencias cgo, es muy fácil de construir y
debería funcionar en cualquier plataforma soportada por Go.

Te da mucho mejor control sobre el entorno de ejecución por lo que puede ser útil para investigación.

### ¿Es seguro para goroutines?

No. Una instancia de goja.Runtime solo puede ser usada por una única goroutine
a la vez. Puedes crear tantas instancias de Runtime como quieras pero
no es posible pasar valores de objetos entre runtimes.

### ¿Dónde está setTimeout()/setInterval()?

setTimeout() y setInterval() son funciones comunes para proporcionar ejecución concurrente en entornos ECMAScript, pero las dos funciones no son parte del estándar ECMAScript.
Los navegadores y NodeJS simplemente proporcionan funciones similares, pero no idénticas. La aplicación host necesita controlar el entorno para ejecución concurrente, ej. un event loop, y suministrar la funcionalidad al código script.

Hay un [proyecto separado](https://github.com/dop251/goja_nodejs) dirigido a proporcionar alguna funcionalidad NodeJS,
e incluye un event loop.

### ¿Puedes implementar (característica X de ES6 o superior)?

Estaré agregando características en su orden de dependencia y tan rápido como el tiempo lo permita. Por favor no preguntes
por ETAs. Las características que están abiertas en el [milestone](https://github.com/dop251/goja/milestone/1) están en progreso
o se trabajarán a continuación.

El trabajo en curso se hace en ramas de características separadas que se fusionan en master cuando es apropiado.
Cada commit en estas ramas representa un estado relativamente estable (es decir, compila y pasa todas las pruebas tc39 habilitadas),
sin embargo, porque la versión de las pruebas tc39 que uso es bastante antigua, puede no estar tan bien probada como la funcionalidad ES5.1. Porque no hay (generalmente) cambios importantes entre las revisiones de ECMAScript
no debería romper tu código existente. Te animamos a probarlo e informar cualquier error encontrado. Por favor no envíes correcciones sin discutirlo primero, ya que el código podría cambiar mientras tanto.

### ¿Cómo contribuyo?

Antes de enviar un pull request asegúrate de que:

- Seguiste el estándar ECMA lo más cerca posible. Si agregas una nueva característica asegúrate de haber leído la especificación,
no te bases solo en un par de ejemplos que funcionan bien.
- Tu cambio no tiene un impacto negativo significativo en el rendimiento (a menos que sea una corrección de errores y sea inevitable)
- Pasa todas las pruebas tc39 relevantes.

Estado Actual
-------------

 * No debería haber cambios importantes en la API, sin embargo puede ser extendida.
 * Falta algo de la funcionalidad AnnexB.

Ejemplo Básico
--------------

Ejecuta JavaScript y obtén el valor resultado.

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

Pasando Valores a JS
--------------------
Cualquier valor Go puede ser pasado a JS usando el método Runtime.ToValue(). Ver la [documentación](https://pkg.go.dev/github.com/dop251/goja#Runtime.ToValue) del método para más detalles.

Exportando Valores desde JS
---------------------------
Un valor JS puede ser exportado a su representación Go por defecto usando el método Value.Export().

Alternativamente puede ser exportado a una variable Go específica usando el método [Runtime.ExportTo()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ExportTo).

Dentro de una sola operación de exportación el mismo Object será representado por el mismo valor Go (ya sea el mismo map, slice o
un puntero a la misma struct). Esto incluye objetos circulares y hace posible exportarlos.

Llamando funciones JS desde Go
------------------------------
Hay 2 enfoques:

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

fmt.Println(sum(40, 2)) // nota, el valor _this_ en la función será undefined.
// Output: 42
```

El primero es más de bajo nivel y permite especificar el valor _this_, mientras que el segundo hace que la función parezca
una función Go normal.

Mapeando nombres de campos y métodos de struct
-----------------------------------------------
Por defecto, los nombres se pasan tal cual, lo que significa que están en mayúsculas. Esto no coincide con
la convención de nomenclatura estándar de JavaScript, así que si necesitas hacer que tu código JS se vea más natural o si estás
tratando con una biblioteca de terceros, puedes usar un [FieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#FieldNameMapper):

```go
vm := goja.New()
vm.SetFieldNameMapper(TagFieldNameMapper("json", true))
type S struct {
    Field int `json:"field"`
}
vm.Set("s", S{Field: 42})
res, _ := vm.RunString(`s.field`) // sin el mapper habría sido s.Field
fmt.Println(res.Export())
// Output: 42
```

Hay dos mappers estándar: [TagFieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#TagFieldNameMapper) y
[UncapFieldNameMapper](https://pkg.go.dev/github.com/dop251/goja#UncapFieldNameMapper), o puedes usar tu propia implementación.

Constructores Nativos
---------------------

Para implementar una función constructor en Go usa `func (goja.ConstructorCall) *goja.Object`.
Ver la documentación de [Runtime.ToValue()](https://pkg.go.dev/github.com/dop251/goja#Runtime.ToValue) para más detalles.

Expresiones Regulares
---------------------

Goja usa la biblioteca regexp de Go incorporada cuando es posible, de lo contrario recurre a [regexp2](https://github.com/dlclark/regexp2).

Excepciones
-----------

Cualquier excepción lanzada en JavaScript se devuelve como un error de tipo *Exception. Es posible extraer el valor lanzado
usando el método Value():

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

Si una función Go nativa hace panic con un Value, se lanza como una excepción Javascript (y por lo tanto puede ser capturada):

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

Interrumpiendo
--------------

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
    // err es de tipo *InterruptError y su método Value() devuelve lo que se haya pasado a vm.Interrupt()
}
```

Compatibilidad con NodeJS
-------------------------

Hay un [proyecto separado](https://github.com/dop251/goja_nodejs) dirigido a proporcionar algo de la funcionalidad de NodeJS.