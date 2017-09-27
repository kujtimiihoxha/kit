package main

import (
	"context"
	"fmt"
	"os"
	"time"

	grpc3 "github.com/go-kit/kit/transport/grpc"
	grpc2 "github.com/kujtimiihoxha/kit/test_dir/math/client/grpc"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial(":8082", grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
	defer conn.Close()
	svc, err := grpc2.New(conn, map[string][]grpc3.ClientOption{})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(svc.Sum(context.Background(), 2, 3))
	fmt.Println(svc.Prod(context.Background(), 2, 3))
}
