#include <linux/module.h>
#include <linux/export-internal.h>
#include <linux/compiler.h>

MODULE_INFO(name, KBUILD_MODNAME);

__visible struct module __this_module
__section(".gnu.linkonce.this_module") = {
	.name = KBUILD_MODNAME,
	.init = init_module,
#ifdef CONFIG_MODULE_UNLOAD
	.exit = cleanup_module,
#endif
	.arch = MODULE_ARCH_INIT,
};



static const struct modversion_info ____versions[]
__used __section("__versions") = {
	{ 0x7b4041d1, "single_open" },
	{ 0x46968a43, "proc_remove" },
	{ 0xbd03ed67, "__ref_stack_chk_guard" },
	{ 0x976c74cb, "get_task_mm" },
	{ 0xe6cc5085, "__tracepoint_mmap_lock_start_locking" },
	{ 0x8efcc8cd, "down_read" },
	{ 0xe6cc5085, "__tracepoint_mmap_lock_acquire_returned" },
	{ 0xe6cc5085, "__tracepoint_mmap_lock_released" },
	{ 0x8efcc8cd, "up_read" },
	{ 0xe838feb3, "access_process_vm" },
	{ 0x35283b01, "mmput" },
	{ 0x12ca6142, "ktime_get_with_offset" },
	{ 0xeaee2b79, "seq_printf" },
	{ 0xe78e4943, "__mmap_lock_do_trace_released" },
	{ 0x790053b1, "__mmap_lock_do_trace_acquire_returned" },
	{ 0xe78e4943, "__mmap_lock_do_trace_start_locking" },
	{ 0x40a621c5, "snprintf" },
	{ 0x90a48d82, "__ubsan_handle_out_of_bounds" },
	{ 0xd272d446, "__stack_chk_fail" },
	{ 0xc7ffe1aa, "si_meminfo" },
	{ 0xd272d446, "__rcu_read_lock" },
	{ 0x151d4c65, "init_task" },
	{ 0xd272d446, "__rcu_read_unlock" },
	{ 0x1cf09ab5, "__put_task_struct_rcu_cb" },
	{ 0xb9fcd065, "call_rcu" },
	{ 0xff0106da, "refcount_warn_saturate" },
	{ 0x41fd17a1, "seq_read" },
	{ 0x68140f95, "seq_lseek" },
	{ 0x1fde1a17, "single_release" },
	{ 0xd272d446, "__fentry__" },
	{ 0x6b01a33d, "proc_create" },
	{ 0xe8213e80, "_printk" },
	{ 0xd272d446, "__x86_return_thunk" },
	{ 0xd954c786, "module_layout" },
};

static const u32 ____version_ext_crcs[]
__used __section("__version_ext_crcs") = {
	0x7b4041d1,
	0x46968a43,
	0xbd03ed67,
	0x976c74cb,
	0xe6cc5085,
	0x8efcc8cd,
	0xe6cc5085,
	0xe6cc5085,
	0x8efcc8cd,
	0xe838feb3,
	0x35283b01,
	0x12ca6142,
	0xeaee2b79,
	0xe78e4943,
	0x790053b1,
	0xe78e4943,
	0x40a621c5,
	0x90a48d82,
	0xd272d446,
	0xc7ffe1aa,
	0xd272d446,
	0x151d4c65,
	0xd272d446,
	0x1cf09ab5,
	0xb9fcd065,
	0xff0106da,
	0x41fd17a1,
	0x68140f95,
	0x1fde1a17,
	0xd272d446,
	0x6b01a33d,
	0xe8213e80,
	0xd272d446,
	0xd954c786,
};
static const char ____version_ext_names[]
__used __section("__version_ext_names") =
	"single_open\0"
	"proc_remove\0"
	"__ref_stack_chk_guard\0"
	"get_task_mm\0"
	"__tracepoint_mmap_lock_start_locking\0"
	"down_read\0"
	"__tracepoint_mmap_lock_acquire_returned\0"
	"__tracepoint_mmap_lock_released\0"
	"up_read\0"
	"access_process_vm\0"
	"mmput\0"
	"ktime_get_with_offset\0"
	"seq_printf\0"
	"__mmap_lock_do_trace_released\0"
	"__mmap_lock_do_trace_acquire_returned\0"
	"__mmap_lock_do_trace_start_locking\0"
	"snprintf\0"
	"__ubsan_handle_out_of_bounds\0"
	"__stack_chk_fail\0"
	"si_meminfo\0"
	"__rcu_read_lock\0"
	"init_task\0"
	"__rcu_read_unlock\0"
	"__put_task_struct_rcu_cb\0"
	"call_rcu\0"
	"refcount_warn_saturate\0"
	"seq_read\0"
	"seq_lseek\0"
	"single_release\0"
	"__fentry__\0"
	"proc_create\0"
	"_printk\0"
	"__x86_return_thunk\0"
	"module_layout\0"
;

MODULE_INFO(depends, "");


MODULE_INFO(srcversion, "B4FE484070124D040F9E4E4");
