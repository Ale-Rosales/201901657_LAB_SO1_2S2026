// Sonda eBPF del Proyecto 2 SO1 (201901657)
// Se engancha al tracepoint syscalls:sys_enter_kill para auditar cada
// vez que un proceso envia una señal de terminacion a otro. Publica
// los eventos en un ring buffer que el Daemon en Go lee en tiempo
// real, para usarlos como "confirmacion real" de que un contenedor
// fue efectivamente terminado.
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

char LICENSE[] SEC("license") = "GPL";

// Estructura de cada evento que mandamos a espacio de usuario.
struct kill_event {
    __u32 caller_pid; // quien mando la señal (el PID del Daemon, si fue el)
    __u32 target_pid; // quien la recibio (el proceso del contenedor)
    __s32 signal;     // SIGTERM=15, SIGKILL=9, etc.
};

// Ring buffer: la forma moderna y eficiente de mandar eventos del
// kernel a espacio de usuario (reemplaza a los perf buffers viejos).
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024); // 256 KB de buffer
} events SEC(".maps");

// El tracepoint syscalls:sys_enter_kill expone sus argumentos en un
// struct estandar (trace_event_raw_sys_enter, definido en vmlinux.h)
// con la firma real de kill(pid_t pid, int sig):
//   args[0] = pid a señalizar
//   args[1] = señal a enviar
SEC("tracepoint/syscalls/sys_enter_kill")
int handle_sys_enter_kill(struct trace_event_raw_sys_enter *ctx)
{
    struct kill_event *ev;

    ev = bpf_ringbuf_reserve(&events, sizeof(*ev), 0);
    if (!ev)
        return 0; // buffer lleno, se descarta el evento (no hay espacio)

    ev->caller_pid = (__u32)(bpf_get_current_pid_tgid() >> 32);
    ev->target_pid = (__u32)ctx->args[0];
    ev->signal     = (__s32)ctx->args[1];

    bpf_ringbuf_submit(ev, 0);
    return 0;
}
