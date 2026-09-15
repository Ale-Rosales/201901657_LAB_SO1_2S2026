// Package dockerinfo consulta a Docker (via "docker ps") para saber
// que contenedores fueron creados por nuestro cronjob (identificados
// por el label pr2so1=managed) y su perfil de consumo asignado.
package dockerinfo

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ManagedContainer representa un contenedor creado por el cronjob del
// proyecto, identificable por sus labels.
type ManagedContainer struct {
	ID     string // ID corto (12 chars), igual al que usa el correlator
	Name   string
	Perfil string // "alto-ram" | "alto-cpu" | "bajo" | "intruso"
}

// ListManaged devuelve todos los contenedores con label pr2so1=managed
// actualmente corriendo.
func ListManaged() ([]ManagedContainer, error) {
	// --no-trunc para asegurar el ID completo, luego lo recortamos a 12
	// para que coincida con el formato que usa el package correlator.
	cmd := exec.Command("docker", "ps",
		"--filter", "label=pr2so1=managed",
		"--no-trunc",
		"--format", "{{.ID}}|{{.Names}}|{{.Label \"pr2so1-perfil\"}}",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error ejecutando docker ps: %w", err)
	}

	var containers []ManagedContainer
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		id := parts[0]
		if len(id) > 12 {
			id = id[:12]
		}
		containers = append(containers, ManagedContainer{
			ID:     id,
			Name:   parts[1],
			Perfil: parts[2],
		})
	}
	return containers, nil
}

// Kill mata un contenedor completo vía Docker (usado como red de
// seguridad / fallback; el daemon usa preferentemente syscall.Kill
// sobre el PID del host, ver package manager).
func Kill(name string) error {
	cmd := exec.Command("docker", "kill", name)
	return cmd.Run()
}

// perfilSpec replica las 3 imagenes de prueba definidas en el
// enunciado (misma tabla que usa deploy_contenedores.sh), para que el
// daemon pueda "dar de alta" contenedores cuando un bucket cae por
// debajo del minimo requerido.
var perfilSpecs = map[string]struct {
	Imagen string
	Args   []string
}{
	"alto-ram": {Imagen: "roldyoran/go-client"},
	"alto-cpu": {
		Imagen: "alpine",
		Args:   []string{"sh", "-c", "while true; do echo '2^1000000'; done | bc > /dev/null"},
	},
	"bajo": {Imagen: "alpine", Args: []string{"sleep", "240"}},
}

// Launch crea un nuevo contenedor del perfil indicado, con los mismos
// labels que usa el cronjob, para que el resto del sistema (daemon,
// dashboards) lo reconozca igual que a los demas.
func Launch(perfil string) error {
	spec, ok := perfilSpecs[perfil]
	if !ok {
		return fmt.Errorf("perfil desconocido: %s", perfil)
	}

	nombre := fmt.Sprintf("pr2so1-%s-daemon-%d", perfil, timeNowUnix())
	args := []string{"run", "-d", "--name", nombre,
		"--label", "pr2so1=managed",
		"--label", "pr2so1-perfil=" + perfil,
		spec.Imagen,
	}
	args = append(args, spec.Args...)

	cmd := exec.Command("docker", args...)
	return cmd.Run()
}

func timeNowUnix() int64 {
	return time.Now().Unix()
}
