#!/bin/bash
# Script para ejecutar el debugger con la configuración correcta de terminal
# Este script soluciona problemas de compatibilidad con Enter y flechas

# Guardar configuración actual del terminal
OLD_STTY=$(stty -g)

# Función para restaurar la configuración del terminal al salir
cleanup() {
    stty "$OLD_STTY"
    echo ""
    echo "Terminal restaurado."
}

# Configurar trap para limpiar al salir
trap cleanup EXIT INT TERM

# Configurar el terminal para modo raw con echo
# Esto permite que las teclas funcionen correctamente
stty raw -echo

# Ejecutar el debugger
./goja-debug "$@"

# La limpieza se hace automáticamente por el trap