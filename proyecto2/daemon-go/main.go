package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"daemon-pr2-so1/correlator"
	"daemon-pr2-so1/dockerinfo"
	"daemon-pr2-so1/ebpfprobe"
	"daemon-pr2-so1/manager"
	"daemon-pr2-so1/parser"
	"daemon-pr2-so1/store"
)

// --- Configuracion: rutas dentro de la carpeta final del repositorio. ---
const (
	composeDir      = "/home/alejandro/Escritorio/201901657_LAB_SO1_2S2026/proyecto2/grafana"
	cronScriptPath  = "/home/alejandro/Escritorio/201901657_LAB_SO1_2S2026/proyecto2/cronjob/deploy_contenedores.sh"
	kernelModuleDir = "/home/alejandro/Escritorio/201901657_LAB_SO1_2S2026/proyecto2/kernel-module"
	cycleInterval   = 30 * time.Second // dentro del rango 20-60s que pide el enunciado
)

func main() {
	log.Println("=== Daemon Proyecto 2 SO1 (201901657) iniciando ===")

	if err := startGrafana(); err != nil {
		log.Printf("advertencia: no se pudo levantar Grafana via docker compose: %v", err)
	} else {
		log.Println("Grafana (docker compose) levantado.")
	}

	if err := loadKernelModule(); err != nil {
		log.Fatalf("error cargando el modulo de kernel (paso obligatorio): %v", err)
	}
	log.Println("Modulo de kernel cargado.")

	if err := installCronjob(); err != nil {
		log.Printf("advertencia: no se pudo instalar el cronjob: %v", err)
	} else {
		log.Println("Cronjob instalado (cada 1 minuto).")
	}

	st := store.New("localhost:6379")
	defer st.Close()
	if err := st.Ping(); err != nil {
		log.Fatalf("error conectando a Valkey: %v", err)
	}
	log.Println("Conexion a Valkey OK.")

	prober, err := ebpfprobe.Start()
	if err != nil {
		log.Fatalf("error arrancando la sonda eBPF: %v", err)
	}
	defer prober.Close()
	log.Println("Sonda eBPF activa (syscalls:sys_enter_kill).")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(cycleInterval)
	defer ticker.Stop()

	log.Printf("Daemon corriendo. Ciclo cada %s. Ctrl+C para detener.\n", cycleInterval)
	runCycle(st, prober) // primer ciclo inmediato, sin esperar el primer tick

	for {
		select {
		case <-ticker.C:
			runCycle(st, prober)
		case sig := <-sigCh:
			log.Printf("Señal %v recibida, iniciando apagado ordenado...", sig)
			if err := removeCronjob(); err != nil {
				log.Printf("advertencia: no se pudo remover el cronjob: %v", err)
			} else {
				log.Println("Cronjob removido.")
			}
			log.Println("Daemon detenido.")
			return
		}
	}
}

// runCycle ejecuta una pasada completa: leer snapshot, correlacionar,
// loggear en Valkey, decidir y ejecutar altas/bajas. Los errores se
// registran pero NO detienen el daemon (un ciclo fallido no debe
// tumbar el servicio completo).
func runCycle(st *store.Store, prober *ebpfprobe.Prober) {
	log.Println("--- Nuevo ciclo ---")

	snap, err := parser.ReadSnapshot()
	if err != nil {
		log.Printf("error leyendo snapshot del kernel: %v", err)
		return
	}
	correlator.Attach(snap.Processes)

	if err := st.SaveRAMSnapshot(snap.Mem); err != nil {
		log.Printf("advertencia: no se pudo guardar snapshot de RAM: %v", err)
	}

	managed, err := dockerinfo.ListManaged()
	if err != nil {
		log.Printf("error listando contenedores gestionables: %v", err)
		return
	}
	fmt.Printf("Contenedores gestionables: %d\n", len(managed))

	allMetrics := manager.AllMetrics(snap.Processes, managed)
	if err := st.UpdateRankings(allMetrics); err != nil {
		log.Printf("advertencia: no se pudo actualizar rankings: %v", err)
	}

	decisions := manager.Decide(snap.Processes, managed)

	// PIDs que vamos a intentar terminar en este ciclo, para poder
	// pedirle a la sonda eBPF que confirme cuales realmente murieron.
	// Mantenemos tambien el mapeo PID -> indice de decision, porque un
	// contenedor puede tener varios PIDs y basta con que UNO se
	// confirme para considerar el contenedor eliminado.
	esperados := make(map[int]bool)
	pidToDecision := make(map[int]int)
	for i, d := range decisions {
		for _, pid := range d.Container.PIDs {
			esperados[pid] = true
			pidToDecision[pid] = i
		}
	}

	killResults := manager.ExecuteKills(decisions)
	for _, r := range killResults {
		fmt.Println(r.String())
	}

	// Confirmacion real via eBPF (no confiamos solo en que syscall.Kill
	// no haya devuelto error): esperamos hasta 5s a que la sonda
	// reporte los sys_kill correspondientes a los PIDs que atacamos.
	confirmedPIDs := prober.WaitConfirmations(esperados, 5*time.Second)

	confirmedDecisions := make(map[int]bool)
	for pid := range confirmedPIDs {
		if idx, ok := pidToDecision[pid]; ok {
			confirmedDecisions[idx] = true
		}
	}
	numConfirmados := len(confirmedDecisions)

	if numConfirmados < len(decisions) {
		log.Printf("advertencia: %d/%d contenedores sin confirmar por eBPF (puede que ya no existieran o la señal se perdiera)",
			len(decisions)-numConfirmados, len(decisions))
	}
	if err := st.RecordEliminaciones(numConfirmados); err != nil {
		log.Printf("advertencia: no se pudo registrar eliminaciones: %v", err)
	}

	launches := manager.DecideLaunches(snap.Processes, managed)
	launchResults := manager.ExecuteLaunches(launches)
	for _, r := range launchResults {
		if r.Err != nil {
			fmt.Printf("FALLO al lanzar perfil=%s: %v\n", r.Perfil, r.Err)
		} else {
			fmt.Printf("OK: nuevo contenedor perfil=%s lanzado\n", r.Perfil)
		}
	}

	fmt.Printf("Ciclo completo: %d eliminados (confirmados por eBPF), %d altas.\n",
		numConfirmados, len(launchResults))
}

func startGrafana() error {
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = composeDir
	return cmd.Run()
}

func loadKernelModule() error {
	if moduleLoaded() {
		return nil // ya esta cargado (ej. de una corrida anterior), nada que hacer
	}

	cmd := exec.Command("make", "load")
	cmd.Dir = kernelModuleDir
	if err := cmd.Run(); err != nil {
		// "insmod: File exists" puede pasar si se cargo justo entre el
		// chequeo y el intento; verificamos el estado real antes de
		// declarar el error como fatal.
		if moduleLoaded() {
			return nil
		}
		return err
	}
	return nil
}

// moduleLoaded verifica la fuente de verdad real: si el archivo /proc
// del modulo existe, esta cargado (no dependemos de parsear "lsmod").
func moduleLoaded() bool {
	_, err := os.Stat("/proc/continfo_pr2_so1_201901657")
	return err == nil
}

func installCronjob() error {
	linea := fmt.Sprintf("* * * * * %s", cronScriptPath)
	script := fmt.Sprintf(`(crontab -l 2>/dev/null | grep -vF %q; echo %q) | crontab -`,
		cronScriptPath, linea)
	cmd := exec.Command("sh", "-c", script)
	return cmd.Run()
}

func removeCronjob() error {
	script := fmt.Sprintf(`crontab -l 2>/dev/null | grep -vF %q | crontab - || true`, cronScriptPath)
	cmd := exec.Command("sh", "-c", script)
	return cmd.Run()
}
