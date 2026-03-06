package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/parandhamareddybommaka/kube/pkg/agent"
	"github.com/parandhamareddybommaka/kube/pkg/runtime"
	"github.com/parandhamareddybommaka/kube/pkg/store"
)

func main() {
	var (
		nodeName         = flag.String("node-name", "", "node name")
		etcdEndpoint     = flag.String("etcd", "localhost:2379", "etcd endpoint")
		containerdSocket = flag.String("containerd", "/run/containerd/containerd.sock", "containerd socket")
		podCIDR          = flag.String("pod-cidr", "10.244.0.0/24", "pod CIDR")
		useMockRuntime   = flag.Bool("mock-runtime", false, "use mock container runtime")
		useMemStore      = flag.Bool("memory", false, "use in-memory store")
	)
	flag.Parse()

	if *nodeName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get hostname: %v\n", err)
			os.Exit(1)
		}
		*nodeName = hostname
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var st store.Store
	var err error

	if *useMemStore {
		st = store.NewInMemoryStore()
		fmt.Println("using in-memory store")
	} else {
		st, err = store.NewEtcdStore(store.EtcdConfig{
			Endpoints:   []string{*etcdEndpoint},
			DialTimeout: 5 * time.Second,
			KeyPrefix:   "/kube",
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to connect to etcd: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("connected to etcd at %s\n", *etcdEndpoint)
	}
	defer st.Close()

	var rt runtime.ContainerRuntime
	if *useMockRuntime {
		rt = runtime.NewMockRuntime()
		fmt.Println("using mock container runtime")
	} else {
		rt, err = runtime.NewContainerdRuntime(*containerdSocket, "kube")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to connect to containerd: %v\n", err)
			fmt.Println("falling back to mock runtime")
			rt = runtime.NewMockRuntime()
		}
	}

	ag, err := agent.New(agent.AgentConfig{
		NodeName: *nodeName,
		Store:    st,
		Runtime:  rt,
		PodCIDR:  *podCIDR,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create agent: %v\n", err)
		os.Exit(1)
	}

	if err := ag.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "agent failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("agent started on node %s\n", *nodeName)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("shutting down...")
	ag.Stop()
}
