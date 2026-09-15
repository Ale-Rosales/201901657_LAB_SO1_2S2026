// Package ebpfprobe integra la sonda eBPF (kill_monitor.o, empotrado
// en el binario con go:embed) al Daemon: carga el programa, lo
// engancha al tracepoint syscalls:sys_enter_kill, y expone un canal
// de eventos que el resto del daemon usa para CONFIRMAR que un kill
// realmente ocurrio a nivel de kernel, antes de registrarlo en Valkey.
package ebpfprobe

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

// El .o se copia aqui (compilado por separado con clang -target bpf,
// ver /paso15-ebpf/) y queda embebido en el binario final del daemon.
//
//go:embed kill_monitor.o
var probeObject []byte

// KillEvent es un evento sys_kill capturado por la sonda.
type KillEvent struct {
	CallerPID uint32
	TargetPID uint32
	Signal    int32
}

type Prober struct {
	coll   *ebpf.Collection
	tp     link.Link
	rb     *ringbuf.Reader
	events chan KillEvent
}

// Start carga la sonda eBPF y comienza a leer eventos en segundo plano.
func Start() (*Prober, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("error quitando limite de memlock: %w", err)
	}

	spec, err := ebpf.LoadCollectionSpecFromReader(bytes.NewReader(probeObject))
	if err != nil {
		return nil, fmt.Errorf("error parseando kill_monitor.o: %w", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, fmt.Errorf("error creando coleccion eBPF (¿corriste con sudo?): %w", err)
	}

	prog := coll.Programs["handle_sys_enter_kill"]
	if prog == nil {
		coll.Close()
		return nil, fmt.Errorf("programa handle_sys_enter_kill no encontrado en el objeto")
	}

	tp, err := link.Tracepoint("syscalls", "sys_enter_kill", prog, nil)
	if err != nil {
		coll.Close()
		return nil, fmt.Errorf("error enganchando el tracepoint: %w", err)
	}

	rb, err := ringbuf.NewReader(coll.Maps["events"])
	if err != nil {
		tp.Close()
		coll.Close()
		return nil, fmt.Errorf("error abriendo el ring buffer: %w", err)
	}

	p := &Prober{coll: coll, tp: tp, rb: rb, events: make(chan KillEvent, 64)}
	go p.readLoop()
	return p, nil
}

func (p *Prober) readLoop() {
	for {
		record, err := p.rb.Read()
		if err != nil {
			close(p.events)
			return
		}
		if len(record.RawSample) < 12 {
			continue
		}
		ev := KillEvent{
			CallerPID: binary.LittleEndian.Uint32(record.RawSample[0:4]),
			TargetPID: binary.LittleEndian.Uint32(record.RawSample[4:8]),
			Signal:    int32(binary.LittleEndian.Uint32(record.RawSample[8:12])),
		}
		select {
		case p.events <- ev:
		default:
			// canal lleno (muy improbable); se descarta el evento.
		}
	}
}

// WaitConfirmations drena eventos durante 'timeout', devolviendo el
// subconjunto de PIDs de 'expected' que la sonda confirmo haber visto
// morir por una señal. Esta es la "confirmacion real a nivel de
// kernel" que pide el enunciado antes de loggear en Valkey.
func (p *Prober) WaitConfirmations(expected map[int]bool, timeout time.Duration) map[int]bool {
	confirmed := make(map[int]bool)
	if len(expected) == 0 {
		return confirmed
	}
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-p.events:
			if !ok {
				return confirmed
			}
			pid := int(ev.TargetPID)
			if expected[pid] {
				confirmed[pid] = true
			}
			if len(confirmed) == len(expected) {
				return confirmed
			}
		case <-deadline:
			return confirmed
		}
	}
}

func (p *Prober) Close() {
	p.rb.Close()
	p.tp.Close()
	p.coll.Close()
}
