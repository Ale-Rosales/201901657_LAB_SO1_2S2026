#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/mm.h>
#include <linux/sched.h>
#include <linux/sched/signal.h>
#include <linux/sched/mm.h>
#include <linux/math64.h>
#include <linux/jiffies.h>

#define CARNET "201901657"
#define PROC_NAME "continfo_pr2_so1_" CARNET
#define CMDLINE_BUF_SZ 256

MODULE_LICENSE("GPL");
MODULE_AUTHOR("201901657");
MODULE_DESCRIPTION("Proyecto 2 SO1 - Sonda de kernel: RAM + procesos (paso 4)");

/*
 * get_cmdline() no esta exportada para modulos externos en este kernel,
 * asi que replicamos su logica: la linea de comandos vive en el espacio
 * de memoria del proceso, entre mm->arg_start y mm->arg_end, y hay que
 * leerla con access_process_vm (no se puede acceder directo con un
 * puntero porque es memoria de OTRO proceso).
 */
static int read_cmdline(struct task_struct *task, struct mm_struct *mm,
                         char *buf, int buflen)
{
    unsigned long arg_start, arg_end;
    int len;

    mmap_read_lock(mm);
    arg_start = mm->arg_start;
    arg_end = mm->arg_end;
    mmap_read_unlock(mm);

    if (arg_end <= arg_start)
        return 0;

    len = (int)(arg_end - arg_start);
    if (len > buflen - 1)
        len = buflen - 1;
    if (len <= 0)
        return 0;

    return access_process_vm(task, arg_start, buf, len, 0);
}

/*
 * Formato de cada linea de proceso, pensado para que el Daemon en Go
 * lo parsee facilmente con strings.Split(linea, "|"):
 *
 *   PID|NOMBRE|CMD|VSZ_KB|RSS_KB|PORC_MEM|PORC_CPU
 */
static void print_process_line(struct seq_file *m, struct task_struct *task,
                                unsigned long total_ram_kb)
{
    struct mm_struct *mm;
    unsigned long vsz_kb = 0, rss_kb = 0;
    unsigned long porc_mem_x100 = 0; /* porcentaje * 100, para 2 decimales sin floats */
    unsigned long porc_cpu_x100 = 0;
    char cmdline[CMDLINE_BUF_SZ];
    int cmd_len;
    u64 total_time_ns, uptime_ns;

    /* mm != NULL solo en procesos de usuario (no kernel threads) */
    mm = get_task_mm(task);
    if (mm) {
        vsz_kb = mm->total_vm << (PAGE_SHIFT - 10);
        rss_kb = get_mm_rss(mm) << (PAGE_SHIFT - 10);

        cmd_len = read_cmdline(task, mm, cmdline, CMDLINE_BUF_SZ - 1);
        if (cmd_len <= 0) {
            /* procesos sin argumentos visibles (ej. daemons) caen aqui */
            snprintf(cmdline, CMDLINE_BUF_SZ, "[%s]", task->comm);
        } else {
            int i;
            cmdline[cmd_len] = '\0';
            /* get_cmdline separa argumentos con \0; los volvemos espacios */
            for (i = 0; i < cmd_len - 1; i++) {
                if (cmdline[i] == '\0')
                    cmdline[i] = ' ';
            }
        }

        if (total_ram_kb > 0)
            porc_mem_x100 = (rss_kb * 10000) / total_ram_kb;

        mmput(mm);
    } else {
        snprintf(cmdline, CMDLINE_BUF_SZ, "[kernel-thread:%s]", task->comm);
    }

    /*
     * %CPU: tiempo total de CPU consumido por el proceso dividido
     * entre CUANTO TIEMPO LLEVA VIVO EL PROPIO PROCESO (no el sistema
     * completo) - asi es como calculan %CPU herramientas como top/ps.
     * Usar el uptime del sistema como denominador hace que cualquier
     * proceso de vida corta (como nuestros contenedores de prueba)
     * salga siempre en ~0.00%, sin importar que tan intensamente use
     * la CPU, porque el sistema lleva horas encendido.
     */
    total_time_ns = task->utime + task->stime;
    uptime_ns = ktime_get_boottime_ns() - task->start_time;
    if (uptime_ns > 0)
        porc_cpu_x100 = div64_u64(total_time_ns * 10000, uptime_ns);

    seq_printf(m, "%d|%s|%s|%lu|%lu|%lu.%02lu|%lu.%02lu\n",
               task->pid,
               task->comm,
               cmdline,
               vsz_kb,
               rss_kb,
               porc_mem_x100 / 100, porc_mem_x100 % 100,
               porc_cpu_x100 / 100, porc_cpu_x100 % 100);
}

static int continfo_show(struct seq_file *m, void *v)
{
    struct sysinfo si;
    unsigned long total_kb, free_kb, used_kb;
    struct task_struct *task;

    si_meminfo(&si);
    total_kb = si.totalram << (PAGE_SHIFT - 10);
    free_kb  = si.freeram  << (PAGE_SHIFT - 10);
    used_kb  = total_kb - free_kb;

    seq_printf(m, "=== INFORMACION DE MEMORIA RAM ===\n");
    seq_printf(m, "RAM_TOTAL_KB: %lu\n", total_kb);
    seq_printf(m, "RAM_LIBRE_KB: %lu\n", free_kb);
    seq_printf(m, "RAM_USADA_KB: %lu\n", used_kb);
    seq_printf(m, "\n=== PROCESOS (PID|NOMBRE|CMD|VSZ_KB|RSS_KB|PORC_MEM|PORC_CPU) ===\n");

    rcu_read_lock();
    for_each_process(task) {
        /* Pin de referencia para poder soltar el RCU lock mientras
         * leemos cmdline (esa operacion puede dormir). */
        get_task_struct(task);
        rcu_read_unlock();

        print_process_line(m, task, total_kb);

        put_task_struct(task);
        rcu_read_lock();
    }
    rcu_read_unlock();

    return 0;
}

static int continfo_open(struct inode *inode, struct file *file)
{
    return single_open(file, continfo_show, NULL);
}

static const struct proc_ops continfo_fops = {
    .proc_open    = continfo_open,
    .proc_read    = seq_read,
    .proc_lseek   = seq_lseek,
    .proc_release = single_release,
};

static struct proc_dir_entry *continfo_entry;

static int __init continfo_init(void)
{
    continfo_entry = proc_create(PROC_NAME, 0444, NULL, &continfo_fops);
    if (!continfo_entry) {
        printk(KERN_ERR "continfo_pr2_so1: fallo al crear /proc/%s\n", PROC_NAME);
        return -ENOMEM;
    }
    printk(KERN_INFO "continfo_pr2_so1: modulo cargado, /proc/%s creado\n", PROC_NAME);
    return 0;
}

static void __exit continfo_exit(void)
{
    proc_remove(continfo_entry);
    printk(KERN_INFO "continfo_pr2_so1: modulo descargado, /proc/%s eliminado\n", PROC_NAME);
}

module_init(continfo_init);
module_exit(continfo_exit);
