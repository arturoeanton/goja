// Ejemplo para probar el debugger de Goja
// Este script demuestra las capacidades del debugger

// Variables globales
var contadorGlobal = 0;
let mensajeGlobal = "Hola desde el debugger!";
const PI_PERSONALIZADO = 3.14159;

// Función para calcular factorial
function factorial(n) {
    // Variables locales que se pueden inspeccionar
    let resultado = 1;
    let contador = n;
    
    console.log("Calculando factorial de " + n);
    
    // Loop que se puede debuggear paso a paso
    while (contador > 1) {
        resultado = resultado * contador;
        contador--;
    }
    
    return resultado;
}

// Función con closure para probar variables en diferentes scopes
function crearContador(inicial) {
    let valor = inicial;
    
    // Función interna que accede a la variable del closure
    function incrementar() {
        valor++;
        console.log("Contador incrementado a: " + valor);
        return valor;
    }
    
    return incrementar;
}

// Función principal
function main() {
    console.log("=== Iniciando programa de ejemplo ===");
    
    // Probar factorial
    let fact5 = factorial(5);
    console.log("Factorial de 5 = " + fact5);
    
    // Probar closure
    let miContador = crearContador(10);
    miContador();
    miContador();
    
    // Objeto para inspeccionar
    let persona = {
        nombre: "Juan",
        edad: 25,
        saludar: function() {
            console.log("Hola, soy " + this.nombre);
        }
    };
    
    persona.saludar();
    
    // Array para inspeccionar
    let numeros = [1, 2, 3, 4, 5];
    let suma = 0;
    for (let i = 0; i < numeros.length; i++) {
        suma += numeros[i];
    }
    console.log("Suma del array: " + suma);
    
    console.log("=== Programa terminado ===");
}

// Ejecutar el programa principal
main();