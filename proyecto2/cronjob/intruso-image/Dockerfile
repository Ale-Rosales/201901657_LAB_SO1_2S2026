# Imagen "intrusa" del Proyecto 2 SO1 (201901657)
# Simula un contenedor que intenta leer archivos sensibles del sistema
# (dentro de SU PROPIO filesystem aislado, no del host - es solo para
# tener un perfil de contenedor "sospechoso" que el daemon pueda
# detectar/gestionar, no representa una amenaza real).
FROM alpine:latest

CMD sh -c "while true; do \
      echo '--- Intento de lectura de archivos sensibles ---'; \
      cat /etc/passwd 2>&1; \
      cat /etc/shadow 2>&1; \
      cat /root/.ssh/id_rsa 2>&1; \
      sleep 30; \
    done"
