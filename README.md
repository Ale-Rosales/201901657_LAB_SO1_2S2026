# Desarrollo, Conexion y Gestión de Contenedores en Entornos Virtualizados


## 1. Instalación de KVM/QEMU y libvirt

```bash:
sudo apt update
sudo apt install -y qemu-kvm libvirt-daemon-system libvirt-clients \
  bridge-utils virt-manager virtinst genisoimage
sudo systemctl enable --now libvirtd
sudo usermod -aG libvirt,kvm $USER
```

## 2. Imagen base y discos por VM

```bash:
wget https://cloud-images.ubuntu.com/releases/24.04/release/\
ubuntu-24.04-server-cloudimg-amd64.img -O ubuntu-base.img

qemu-img create -f qcow2 -F qcow2 -b ubuntu-base.img vm1.qcow2 10G
qemu-img create -f qcow2 -F qcow2 -b ubuntu-base.img vm2.qcow2 10G
qemu-img create -f qcow2 -F qcow2 -b ubuntu-base.img vm3.qcow2 10G
```

## 3. Configuración inicial con cloud-init

```bash:
# Ejemplo para VM1 (se repite el patrón para VM2 y VM3)
cat > vm1-user-data.yaml << 'EOF'
#cloud-config
hostname: vm1-containerd
users:
  - name: alejandro
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: sudo
    shell: /bin/bash
    lock_passwd: false
chpasswd:
  list: |
    alejandro:cambiar123
  expire: false
ssh_pwauth: true
EOF

genisoimage -output vm1-cidata.iso -volid cidata -joliet -rock \
  -graft-points user-data=vm1-user-data.yaml meta-data=vm1-meta-data.yaml
```

## 4. Creación de las VMs

```bash:
virt-install \
  --name vm1-containerd \
  --memory 1536 --vcpus 1 \
  --disk path=~/vms/vm1.qcow2,bus=virtio \
  --disk path=~/vms/vm1-cidata.iso,device=cdrom,bus=sata \
  --os-variant ubuntu24.04 \
  --network network=default,model=virtio \
  --graphics none --console pty,target_type=serial \
  --import --noautoconsole
```

Ya configurado el entorno, se procede a empezar con las vm, una por una.

### 4.1 VM1 - Containerd

```bash:
sudo apt update
sudo apt install -y containerd
sudo systemctl status containerd
```

### 4.2 VM2 - Podman

```bash:
sudo apt update
sudo apt install -y podman
podman --version
```

### 4.3 VM3 - Docker

```bash:
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
```

## 5. Comandos básicos para VMs

### 5.1 Encender las 3 VMs

```bash:
virsh start vm1-containerd
virsh start vm2-podman
virsh start vm3-docker
```

### 5.2 Verificar que las 3 están corriendo

```bash:
virsh list --all
```

### 5.3 Entrar en cada VM

```bash:
virsh console vm1-containerd
virsh console vm2-podman
virsh console vm3-docker
```

El usuario que se creo en las 3 VMs es el mismo.

Usuario: alejandro

Contraseña: cambiar123
