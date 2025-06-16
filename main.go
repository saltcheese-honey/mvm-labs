package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"flag"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	vmID         string
	tapIface     string
	ipAddr       string
	gateway      string
	netmask      string
	initPath     string
	kernelPath   string
	rootfsPath   string
	writablePath string
	runnerToken  string
)

func main() {
	ctx := context.Background()
	// Set default values
	vmID = "firecracker-runner-vm-" + uuid.New().String()[:8]
	tapIface = "ftap0"
	ipAddr = "172.16.0.2"
	gateway = "172.16.0.1"
	netmask = "255.255.255.0"
	initPath = "sbin/init"

	logger := logrus.New().WithField("vm", vmID)

	// Parse command line arguments
	flag.StringVar(&kernelPath, "kernel", "./vmlinux-6.1.128", "Path to kernel image")
	flag.StringVar(&rootfsPath, "rootfs", "./rootfs.ext4", "Path to rootfs image")
	flag.StringVar(&writablePath, "writable", "./writable.img", "Path to writable image")
	flag.StringVar(&runnerToken, "runner-token", "github-runner-firecracker", "GitHub runner token")
	flag.StringVar(&tapIface, "tap-iface", tapIface, "Tap interface name")
	flag.StringVar(&ipAddr, "ip-addr", ipAddr, "IP address")
	flag.StringVar(&gateway, "gateway", gateway, "Gateway")
	flag.StringVar(&netmask, "netmask", netmask, "Netmask")
	flag.StringVar(&initPath, "init-path", initPath, "Init path")
	flag.Parse()

	// Convert paths to absolute
	kernelPath, _ = filepath.Abs(kernelPath)
	rootfsPath, _ = filepath.Abs(rootfsPath)
	writablePath, _ = filepath.Abs(writablePath)

	// Create writable image
	os.Remove(writablePath)
	file, _ := os.Create(writablePath)
	file.Close()
	os.Truncate(writablePath, 4*1024*1024*1024) // 4GB
	os.Chmod(writablePath, 0666)

	instanceMetadata := map[string]string{
		"RUNNER_NAME": "firecracker-runner" + vmID,
		"RUNNER_PAT":  runnerToken,
	}

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
		AllowMMDS: true,
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
		MmdsVersion: firecracker.MMDSv1,
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

	if err := m.SetMetadata(ctx, instanceMetadata); err != nil {
		log.Fatalf("failed to set MMDS metadata: %v", err)
	}

	if err := m.Wait(ctx); err != nil {
		log.Printf("VM exited: %v", err)
	} else {
		log.Printf("VM terminated")
		m.StopVMM()
	}
}
