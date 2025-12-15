package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/pkg/abfclient"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("commands: check, reset, add-whitelist, add-blacklist, remove-whitelist, remove-blacklist")
		return
	}

	cmd := os.Args[1]
	addr := os.Getenv("ABF_GRPC_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := abfclient.New(addr)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	switch cmd {
	case "check":
		login := flag.String("login", "", "")
		pass := flag.String("password", "", "")
		ip := flag.String("ip", "", "")
		flag.CommandLine.Parse(os.Args[2:])
		ok, err := client.Check(ctx, *login, *pass, *ip)
		if err != nil {
			panic(err)
		}
		fmt.Println("ok =", ok)

	case "reset":
		login := flag.String("login", "", "")
		ip := flag.String("ip", "", "")
		flag.CommandLine.Parse(os.Args[2:])
		err := client.ResetBucket(ctx, *login, *ip)
		if err != nil {
			panic(err)
		}
		fmt.Println("reset done")

	case "add-whitelist":
		cidr := flag.String("cidr", "", "")
		flag.CommandLine.Parse(os.Args[2:])
		err := client.AddToWhitelist(ctx, *cidr)
		if err != nil {
			panic(err)
		}
		fmt.Println("added to whitelist")

		// TODO: add commands to manage subnet lists

	default:
		fmt.Println("unknown command:", cmd)
	}
}
