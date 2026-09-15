obj-m += continfo.o

KDIR := /lib/modules/$(shell uname -r)/build
PWD := $(shell pwd)

all:
	make -C $(KDIR) M=$(PWD) modules

clean:
	make -C $(KDIR) M=$(PWD) clean

sign: continfo.ko
	sudo $(KDIR)/scripts/sign-file sha256 \
		/var/lib/shim-signed/mok/MOK.priv \
		/var/lib/shim-signed/mok/MOK.der \
		continfo.ko

continfo.ko:
	$(MAKE) all

load: sign
	sudo insmod continfo.ko

unload:
	sudo rmmod continfo
