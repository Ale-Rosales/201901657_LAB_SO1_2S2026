CLANG := clang
ARCH := x86

kill_monitor.o: kill_monitor.c vmlinux.h
	$(CLANG) -g -O2 -target bpf -D__TARGET_ARCH_$(ARCH) \
		-I. \
		-c kill_monitor.c -o kill_monitor.o

clean:
	rm -f kill_monitor.o
