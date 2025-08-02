// Test de closures para el debugger de Goja
// =========================================
// NOTA: El debugger tiene una limitación con step-into en closures.
// Este archivo demuestra cómo trabajar con esta limitación.

console.log("=== Test de Closures con Debugger ===");

function crearSumador(inicial) {
    console.log("Creando sumador con valor inicial:", inicial);
    let total = inicial;
    
    // IMPORTANTE: Para debuggear esta función closure:
    // 1. Establecer un breakpoint aquí con: b 15
    // 2. Usar step-over en la línea 27, el debugger se detendrá en el breakpoint
    function sumar(cantidad) {
        // BREAKPOINT RECOMENDADO: Línea 15
        console.log("Entrando a sumar con cantidad:", cantidad);
        total += cantidad;
        console.log("Total ahora es:", total);
        return total;
    }
    
    return sumar;
}

// Crear una función closure
let miSumador = crearSumador(100);

// LIMITACIÓN: Step-into NO funcionará aquí
// SOLUCIÓN: Usar breakpoint en línea 15 y continuar
miSumador(10);  // Línea 29 - Step-into no entrará automáticamente
miSumador(20);  // Línea 30

// Ejemplo más complejo con objeto closure
function crearContador() {
    let count = 0;
    
    return {
        incrementar: function() {
            // BREAKPOINT RECOMENDADO: Línea 39
            count++;
            console.log("Contador incrementado a:", count);
            return count;
        },
        
        decrementar: function() {
            // BREAKPOINT RECOMENDADO: Línea 46
            count--;
            console.log("Contador decrementado a:", count);
            return count;
        },
        
        obtener: function() {
            // BREAKPOINT RECOMENDADO: Línea 53
            console.log("Valor actual del contador:", count);
            return count;
        }
    };
}

let contador = crearContador();

// Estas llamadas no funcionarán con step-into
// Usar breakpoints en líneas 39, 46 y 53
contador.incrementar();  // Línea 63
contador.incrementar();  // Línea 64
contador.decrementar();  // Línea 65
let valorFinal = contador.obtener();  // Línea 66

// Comparación: función normal donde step-into SÍ funciona
function funcionNormal(mensaje) {
    console.log("Función normal:", mensaje);
    return mensaje.toUpperCase();
}

// Aquí step-into SÍ funcionará correctamente
let resultado = funcionNormal("hola mundo");  // Línea 75
console.log("Resultado:", resultado);

console.log("\n=== Fin del test ===");
console.log("Valor final del contador:", valorFinal);

// INSTRUCCIONES DE USO:
// ====================
// 1. Ejecutar: ./goja-debug test_closure.js
// 2. Establecer breakpoints:
//    b 15  (función sumar)
//    b 39  (incrementar)
//    b 46  (decrementar)
//    b 53  (obtener)
// 3. Usar 'c' para continuar hasta los breakpoints
// 4. Una vez en el breakpoint, usar 'n' para step-over
// 5. Usar 'locals' para ver las variables del closure