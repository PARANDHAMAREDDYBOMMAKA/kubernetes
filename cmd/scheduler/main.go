package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/parandhamareddybommaka/kube/pkg/scheduler"
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

	sched := scheduler.New(scheduler.SchedulerConfig{
		Store: st,
	})

	if err := sched.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "scheduler failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("scheduler started")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("shutting down...")
	sched.Stop()
}
