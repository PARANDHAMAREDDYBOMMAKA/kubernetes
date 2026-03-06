package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/parandhamareddybommaka/kube/pkg/controller"
	"github.com/parandhamareddybommaka/kube/pkg/store"
)

func main() {
	var (
		etcdEndpoint = flag.String("etcd", "localhost:2379", "etcd endpoint")
		useMemStore  = flag.Bool("memory", false, "use in-memory store")
	)
	flag.Parse()

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

	mgr := controller.NewManager(st)
	if err := mgr.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "controller manager failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("controller manager started")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("shutting down...")
	mgr.Stop()
}
