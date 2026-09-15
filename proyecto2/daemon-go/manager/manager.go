// Package manager agrega las metricas de procesos por contenedor y
// decide, segun las restricciones del enunciado, cuales eliminar.
package manager

import (
	"sort"

	"daemon-pr2-so1/dockerinfo"
	"daemon-pr2-so1/parser"
)

const (
	MinAltoConsumo = 2 // siempre deben existir 2 contenedores de alto consumo
	MinBajoConsumo = 3 // siempre deben existir 3 contenedores de bajo consumo
)

// ContainerMetrics agrega las metricas de todos los procesos que
// pertenecen a un mismo contenedor.
type ContainerMetrics struct {
	ID      string
	Name    string
	Perfil  string
	RssKB   uint64
	VszKB   uint64
	PorcCPU float64
	// PIDs son TODOS los procesos del host que pertenecen a este
	// contenedor (no solo el "principal"). Un contenedor como
	// alto-cpu genera hijos efimeros (bc, sleep) constantemente, y
	// asumir que el PID mas bajo es siempre el proceso principal es
	// una heuristica fragil: si el contador de PIDs del kernel dio la
	// vuelta (pid_max) durante las horas que el sistema lleva
	// corriendo, un hijo recien creado puede terminar con un numero
	// de PID mas bajo que el proceso padre de larga duracion. Por eso
	// guardamos TODOS los PIDs y los matamos todos, en vez de adivinar
	// cual es "el" PID representativo.
	PIDs []int
}

// Decision es el resultado del analisis: que contenedores hay que
// eliminar y por que razon (para logging/manual tecnico).
type Decision struct {
	Container ContainerMetrics
	Razon     string
}

// aggregate agrupa los procesos por ContainerID y suma sus metricas.
func aggregate(processes []parser.Process, managed []dockerinfo.ManagedContainer) map[string]*ContainerMetrics {
	// Mapa auxiliar id -> (name, perfil) para no tener que cruzar listas repetidamente.
	info := make(map[string]dockerinfo.ManagedContainer, len(managed))
	for _, m := range managed {
		info[m.ID] = m
	}

	result := make(map[string]*ContainerMetrics)
	for _, p := range processes {
		if p.ContainerID == "" {
			continue
		}
		m, ok := info[p.ContainerID]
		if !ok {
			// Proceso de un contenedor que NO es nuestro (ej. Grafana,
			// Valkey, Portainer) - lo ignoramos, el daemon no lo gestiona.
			continue
		}

		cm, exists := result[p.ContainerID]
		if !exists {
			cm = &ContainerMetrics{
				ID:     p.ContainerID,
				Name:   m.Name,
				Perfil: m.Perfil,
			}
			result[p.ContainerID] = cm
		}
		cm.RssKB += p.RssKB
		cm.VszKB += p.VszKB
		cm.PorcCPU += p.PorcCPU
		cm.PIDs = append(cm.PIDs, p.PID)
	}
	return result
}

// LaunchAction indica cuantos contenedores de un perfil hay que crear
// para volver a cumplir el minimo requerido.
type LaunchAction struct {
	Perfil   string
	Cantidad int
	Razon    string
}

// AllMetrics agrega las metricas de todos los contenedores gestionados
// actualmente vivos (para actualizar los rankings de Top RAM/CPU en
// cada ciclo, no solo los que se eliminan).
func AllMetrics(processes []parser.Process, managed []dockerinfo.ManagedContainer) []ContainerMetrics {
	metrics := aggregate(processes, managed)
	result := make([]ContainerMetrics, 0, len(metrics))
	for _, cm := range metrics {
		result = append(result, *cm)
	}
	return result
}

// Decide analiza el snapshot actual y devuelve la lista de
// contenedores que deben eliminarse, en el orden en que deben matarse.
func Decide(processes []parser.Process, managed []dockerinfo.ManagedContainer) []Decision {
	metrics := aggregate(processes, managed)

	var intrusos, altos, bajos []*ContainerMetrics
	for _, cm := range metrics {
		switch cm.Perfil {
		case "intruso":
			intrusos = append(intrusos, cm)
		case "alto-ram", "alto-cpu":
			altos = append(altos, cm)
		case "bajo":
			bajos = append(bajos, cm)
		}
	}

	var decisions []Decision

	// Los intrusos se eliminan siempre.
	for _, cm := range intrusos {
		decisions = append(decisions, Decision{
			Container: *cm,
			Razon:     "contenedor intruso detectado",
		})
	}

	decisions = append(decisions, pruneExcess(altos, MinAltoConsumo, "alto consumo")...)
	decisions = append(decisions, pruneExcess(bajos, MinBajoConsumo, "bajo consumo")...)

	return decisions
}

// DecideLaunches calcula que perfiles estan por debajo del minimo
// requerido y cuantos contenedores hay que "dar de alta" para
// restablecer la restriccion de minimos del enunciado.
func DecideLaunches(processes []parser.Process, managed []dockerinfo.ManagedContainer) []LaunchAction {
	metrics := aggregate(processes, managed)

	altoCount, bajoCount := 0, 0
	for _, cm := range metrics {
		switch cm.Perfil {
		case "alto-ram", "alto-cpu":
			altoCount++
		case "bajo":
			bajoCount++
		}
	}

	var launches []LaunchAction
	if altoCount < MinAltoConsumo {
		launches = append(launches, LaunchAction{
			Perfil:   "alto-cpu", // se puede lanzar cualquiera de los 2 perfiles "alto"
			Cantidad: MinAltoConsumo - altoCount,
			Razon:    "por debajo del minimo de alto consumo",
		})
	}
	if bajoCount < MinBajoConsumo {
		launches = append(launches, LaunchAction{
			Perfil:   "bajo",
			Cantidad: MinBajoConsumo - bajoCount,
			Razon:    "por debajo del minimo de bajo consumo",
		})
	}
	return launches
}

// pruneExcess ordena un bucket de contenedores por consumo (RSS
// primero, %CPU como desempate) de mayor a menor, y va marcando para
// eliminar los de mayor consumo mientras el conteo se mantenga por
// encima del minimo requerido.
func pruneExcess(bucket []*ContainerMetrics, minimo int, etiqueta string) []Decision {
	sort.Slice(bucket, func(i, j int) bool {
		if bucket[i].RssKB != bucket[j].RssKB {
			return bucket[i].RssKB > bucket[j].RssKB
		}
		return bucket[i].PorcCPU > bucket[j].PorcCPU
	})

	var decisions []Decision
	restantes := len(bucket)
	for _, cm := range bucket {
		if restantes <= minimo {
			break
		}
		decisions = append(decisions, Decision{
			Container: *cm,
			Razon:     "excede el minimo de " + etiqueta + " (mayor consumidor del grupo)",
		})
		restantes--
	}
	return decisions
}
