#!/bin/bash
# Script de cronjob del Proyecto 2 SO1 (201901657)
# Despliega 5 contenedores aleatorios entre los perfiles de carga de
# trabajo definidos en el enunciado, mas 1 contenedor "intruso".
# Todos llevan labels para que el Daemon en Go los identifique como
# "gestionables" y sepa a que perfil pertenecen.

set -euo pipefail

CARNET="201901657"
LABEL_MANAGED="pr2so1=managed"
INTRUSO_IMG="intruso-pr2-${CARNET}:latest"

# Cada entrada: "perfil|imagen|comando"
PERFILES=(
  "alto-ram|roldyoran/go-client|"
  "alto-cpu|alpine|sh -c \"while true; do echo '2^1000000'; done | bc > /dev/null\""
  "bajo|alpine|sleep 240"
)

lanzar_contenedor() {
  local perfil="$1"
  local imagen="$2"
  local cmd="$3"
  local nombre="pr2so1-${perfil}-$(date +%s)-${RANDOM}"

  echo "Lanzando contenedor perfil=${perfil} imagen=${imagen} nombre=${nombre}"

  if [ -z "$cmd" ]; then
    docker run -d --name "$nombre" \
      --label "$LABEL_MANAGED" \
      --label "pr2so1-perfil=${perfil}" \
      "$imagen"
  else
    eval docker run -d --name "$nombre" \
      --label "$LABEL_MANAGED" \
      --label "pr2so1-perfil=${perfil}" \
      "$imagen" $cmd
  fi
}

# --- Despliegue aleatorio de 5 contenedores ---
for i in $(seq 1 5); do
  idx=$((RANDOM % ${#PERFILES[@]}))
  IFS='|' read -r perfil imagen cmd <<< "${PERFILES[$idx]}"
  lanzar_contenedor "$perfil" "$imagen" "$cmd"
done

# --- Contenedor intruso ---
nombre_intruso="pr2so1-intruso-$(date +%s)-${RANDOM}"
echo "Lanzando contenedor intruso nombre=${nombre_intruso}"
docker run -d --name "$nombre_intruso" \
  --label "$LABEL_MANAGED" \
  --label "pr2so1-perfil=intruso" \
  "$INTRUSO_IMG"

echo "Cronjob completado: 5 contenedores aleatorios + 1 intruso lanzados."
