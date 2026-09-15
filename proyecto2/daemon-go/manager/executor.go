package manager

import (
	"fmt"
	"syscall"
	"time"

	"daemon-pr2-so1/dockerinfo"
)

// KillResult registra el resultado de intentar eliminar un contenedor,
// para poder loggearlo despues (consola y/o Valkey).
type KillResult struct {
	Decision Decision
	Senal    string // "SIGTERM" o "SIGKILL", cual hizo falta para terminarlo
	Err      error
}

// gracePeriod es cuanto esperamos despues de SIGTERM antes de escalar
// a SIGKILL si el proceso sigue vivo.
const gracePeriod = 3 * time.Second

// processAlive verifica si un PID sigue existiendo, enviando la
// "señal 0" (no manda ninguna señal real, solo comprueba permisos y
// existencia). Es el patron estandar en Unix para este chequeo.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}

// ExecuteKills envia SIGTERM directamente a TODOS los PIDs del host
// que pertenecen a cada contenedor en la lista de decisiones (asi la
// sonda eBPF ve el syscall sys_kill saliendo del propio proceso del
// Daemon). Tras un periodo de gracia, cualquier PID que siga vivo
// -- comun cuando el proceso principal es el PID 1 de su namespace y
// no maneja SIGTERM -- se remata con SIGKILL, igual que hace
// "docker stop" internamente. Enviar la señal a todos los PIDs (no
// solo al "principal") evita depender de heuristicas fragiles para
// adivinar cual es el proceso raiz del contenedor.
func ExecuteKills(decisions []Decision) []KillResult {
	results := make([]KillResult, 0, len(decisions))
	for _, d := range decisions {
		pids := d.Container.PIDs

		for _, pid := range pids {
			// Ignoramos el error individual aqui: es normal que un
			// hijo efimero (bc, sleep) ya no exista para cuando
			// llegamos a esta linea.
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}

		time.Sleep(gracePeriod)

		senal := "SIGTERM"
		var ultimoErr error
		vivos := 0
		for _, pid := range pids {
			if !processAlive(pid) {
				continue
			}
			vivos++
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
				ultimoErr = err
			}
		}
		if vivos > 0 {
			senal = "SIGTERM+SIGKILL"
		}

		results = append(results, KillResult{Decision: d, Senal: senal, Err: ultimoErr})
	}
	return results
}

// LaunchResult registra el resultado de intentar dar de alta un
// contenedor de un perfil especifico.
type LaunchResult struct {
	Perfil string
	Err    error
}

// ExecuteLaunches crea, vía Docker, los contenedores necesarios para
// cubrir los faltantes detectados por DecideLaunches.
func ExecuteLaunches(launches []LaunchAction) []LaunchResult {
	var results []LaunchResult
	for _, l := range launches {
		for i := 0; i < l.Cantidad; i++ {
			err := dockerinfo.Launch(l.Perfil)
			results = append(results, LaunchResult{Perfil: l.Perfil, Err: err})
		}
	}
	return results
}

func (r KillResult) String() string {
	if r.Err != nil {
		return fmt.Sprintf("FALLO al eliminar %s (PIDs=%v): %v",
			r.Decision.Container.Name, r.Decision.Container.PIDs, r.Err)
	}
	return fmt.Sprintf("OK: %s (PIDs=%v) eliminado con %s",
		r.Decision.Container.Name, r.Decision.Container.PIDs, r.Senal)
}
