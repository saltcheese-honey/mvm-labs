package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/sirupsen/logrus"
)

const (
	vmID     = "demo-vm"
	tapIface = "ftap0"
	ipAddr   = "172.16.0.2"
	gateway  = "172.16.0.1"
	netmask  = "255.255.255.0"
	initPath = "sbin/init"
)

func main() {
	ctx := context.Background()
	logger := logrus.New().WithField("vm", vmID)

	// Get absolute paths
	kernelPath, _ := filepath.Abs("./vmlinux-6.1.128")
	rootfsPath, _ := filepath.Abs("./rootfs.ext4")
	writablePath, _ := filepath.Abs("./writable.img")

	// Create writable image
	os.Remove(writablePath)
	file, _ := os.Create(writablePath)
	file.Close()
	os.Truncate(writablePath, 4*1024*1024*1024) // 4GB
	os.Chmod(writablePath, 0666)

	// Configure drives
	rootDrive := models.Drive{
		DriveID:      firecracker.String("rootfs"),
		PathOnHost:   firecracker.String(rootfsPath),
		IsRootDevice: firecracker.Bool(true),
		IsReadOnly:   firecracker.Bool(true),
	}

	writableDrive := models.Drive{
		DriveID:      firecracker.String("writable"),
		PathOnHost:   firecracker.String(writablePath),
		IsRootDevice: firecracker.Bool(false),
		IsReadOnly:   firecracker.Bool(false),
	}

	// Configure network
	networkInterface := firecracker.NetworkInterface{
		StaticConfiguration: &firecracker.StaticNetworkConfiguration{
			HostDevName: tapIface,
			MacAddress:  "06:00:AC:10:00:02",
		},
	}

	// Configure machine
	cfg := firecracker.Config{
		SocketPath:      "./" + vmID + ".socket",
		KernelImagePath: kernelPath,
		KernelArgs:      "console=ttyS0 reboot=k panic=1 pci=off ip=172.16.0.2::172.16.0.1:255.255.255.0::eth0 rw init=" + initPath,
		Drives:          []models.Drive{rootDrive, writableDrive},
		NetworkInterfaces: firecracker.NetworkInterfaces{
			networkInterface,
		},
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(2),
			MemSizeMib: firecracker.Int64(4096),
		},
	}

	machineOpts := []firecracker.Opt{
		firecracker.WithLogger(logger),
	}

	m, err := firecracker.NewMachine(ctx, cfg, machineOpts...)
	if err != nil {
		log.Fatalf("failed creating machine: %s", err)
	}

	if err := m.Start(ctx); err != nil {
		log.Fatalf("failed starting machine: %s", err)
	}

	if err := m.Wait(ctx); err != nil {
		log.Printf("VM exited: %v", err)
	} else {
		log.Printf("VM terminated")
		m.StopVMM()
	}
}
