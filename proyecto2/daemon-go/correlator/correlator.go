// Package correlator determina a que contenedor Docker pertenece cada
// proceso, leyendo /proc/<pid>/cgroup (funciona tanto con cgroup v1 como
// con cgroup v2 / systemd, que son los dos formatos mas comunes).
package correlator

import (
	"os"
	"regexp"
	"strconv"

	"daemon-pr2-so1/parser"
)

// Un ID de contenedor Docker es un hash hexadecimal de 64 caracteres.
// Aparece en el cgroup como.../docker/<hash> (cgroupfs) o
// .../docker-<hash>.scope (systemd).
var containerIDRegex = regexp.MustCompile(`[0-9a-f]{64}`)

// Attach recorre la lista de procesos y llena ContainerID cuando el
// proceso pertenece a un contenedor. Los procesos que no pertenecen a
// ningun contenedor (procesos normales del sistema) quedan con
// ContainerID vacio.
func Attach(processes []parser.Process) {
	for i := range processes {
		id := containerIDFor(processes[i].PID)
		if id != "" {
			// Usamos el ID corto (12 caracteres), igual que "docker ps".
			processes[i].ContainerID = id[:12]
		}
	}
}

func containerIDFor(pid int) string {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cgroup")
	if err != nil {
		// El proceso pudo haber terminado entre la lectura del /proc
		// del kernel y esta correlacion; lo ignoramos sin fallar.
		return ""
	}
	match := containerIDRegex.FindString(string(data))
	return match
}
