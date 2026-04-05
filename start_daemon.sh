#!/bin/bash

# Directorio del proyecto
PROJECT_DIR="/storage/emulated/0/Documents/gorrson"
cd "$PROJECT_DIR" || exit 1

echo "Compilando gorrson..."
go build -o gorrson main.go

if [ $? -ne 0 ]; then
    echo "Error en la compilación. Abortando."
    exit 1
fi
echo "Compilación exitosa."

# Detener cualquier instancia anterior
echo "Deteniendo instancias anteriores de gorrson..."
pkill -f "./gorrson serve" || true
sleep 1

# Asegurar que el directorio de datos existe
mkdir -p data

# Iniciar el servidor en segundo plano con nohup
# Redirigimos la salida de error (stderr) a stdout (2>&1)
# y todo va al archivo nohup.out (o directamente al log de gorrson si usamos la bandera -log-format)
echo "Iniciando gorrson en segundo plano..."
nohup ./gorrson serve --port 7077 --log-format json > nohup.out 2>&1 &

# Guardar el PID (Process ID) para referencia
PID=$!
echo "Gorrson está corriendo con el PID: $PID"
echo "Puedes ver los logs en tiempo real con: tail -f data/gorrson.log"
