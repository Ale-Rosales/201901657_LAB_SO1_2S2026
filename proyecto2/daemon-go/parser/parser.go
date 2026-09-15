// Package parser lee el archivo /proc/continfo_pr2_so1_201901657 generado
// por el modulo de kernel y lo convierte en structs Go faciles de usar.
package parser

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MemInfo contiene los datos de RAM del sistema, tal como los expone
// el modulo de kernel (bloque "=== INFORMACION DE MEMORIA RAM ===").
type MemInfo struct {
	TotalKB uint64
	FreeKB  uint64
	UsedKB  uint64
}

// Process representa una linea del bloque de procesos:
//
//	PID|NOMBRE|CMD|VSZ_KB|RSS_KB|PORC_MEM|PORC_CPU
type Process struct {
	PID       int
	Nombre    string
	Cmd       string
	VszKB     uint64
	RssKB     uint64
	PorcMem   float64
	PorcCPU   float64
	// ContainerID se llena despues, correlacionando con containerd-shim.
	// Queda vacio si el proceso no pertenece a ningun contenedor.
	ContainerID string
}

// Snapshot es el resultado completo de una lectura del archivo /proc.
type Snapshot struct {
	Mem       MemInfo
	Processes []Process
}

const procPath = "/proc/continfo_pr2_so1_201901657"

// ReadSnapshot lee y parsea el archivo /proc del modulo de kernel.
func ReadSnapshot() (Snapshot, error) {
	f, err := os.Open(procPath)
	if err != nil {
		return Snapshot{}, fmt.Errorf("no se pudo abrir %s: %w (¿esta cargado el modulo de kernel?)", procPath, err)
	}
	defer f.Close()

	var snap Snapshot
	scanner := bufio.NewScanner(f)
	inProcessSection := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "==="):
			// Detecta el cambio de seccion por el titulo del bloque.
			if strings.Contains(line, "PROCESOS") {
				inProcessSection = true
			}
			continue
		case strings.HasPrefix(line, "RAM_TOTAL_KB:"):
			snap.Mem.TotalKB = parseUintAfterColon(line)
		case strings.HasPrefix(line, "RAM_LIBRE_KB:"):
			snap.Mem.FreeKB = parseUintAfterColon(line)
		case strings.HasPrefix(line, "RAM_USADA_KB:"):
			snap.Mem.UsedKB = parseUintAfterColon(line)
		case inProcessSection:
			proc, err := parseProcessLine(line)
			if err != nil {
				// Una linea corrupta no debe tumbar todo el parseo;
				// la saltamos y seguimos con las demas.
				continue
			}
			snap.Processes = append(snap.Processes, proc)
		}
	}

	if err := scanner.Err(); err != nil {
		return Snapshot{}, fmt.Errorf("error leyendo %s: %w", procPath, err)
	}

	return snap, nil
}

func parseUintAfterColon(line string) uint64 {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return 0
	}
	val, _ := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
	return val
}

func parseProcessLine(line string) (Process, error) {
	fields := strings.Split(line, "|")

	// El formato tiene 7 campos fijos: PID|NOMBRE|CMD|VSZ|RSS|%MEM|%CPU.
	// PERO el campo CMD es texto libre (la linea de comandos real del
	// proceso) y puede contener el propio caracter "|" -- por ejemplo
	// "sh -c echo 2^20 | bc" literalmente tiene un pipe adentro. Eso
	// hace que la linea completa tenga MAS de 7 campos al dividir por
	// "|". La solucion: los primeros 2 campos (PID, NOMBRE) y los
	// ultimos 4 (VSZ, RSS, %MEM, %CPU) son siempre fijos y nunca
	// contienen "|"; todo lo que sobra en el medio es CMD, y lo
	// reconstruimos uniendolo de nuevo con "|".
	if len(fields) < 7 {
		return Process{}, fmt.Errorf("linea con %d campos, se esperaban al menos 7: %q", len(fields), line)
	}

	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return Process{}, fmt.Errorf("PID invalido: %w", err)
	}

	cmd := strings.Join(fields[2:len(fields)-4], "|")

	tail := fields[len(fields)-4:]
	vsz, _ := strconv.ParseUint(tail[0], 10, 64)
	rss, _ := strconv.ParseUint(tail[1], 10, 64)
	porcMem, _ := strconv.ParseFloat(tail[2], 64)
	porcCPU, _ := strconv.ParseFloat(tail[3], 64)

	return Process{
		PID:     pid,
		Nombre:  fields[1],
		Cmd:     cmd,
		VszKB:   vsz,
		RssKB:   rss,
		PorcMem: porcMem,
		PorcCPU: porcCPU,
	}, nil
}
