package main

import (
	"context"
	"fmt"

	http3 "github.com/go-kit/kit/transport/http"
	"github.com/kujtimiihoxha/kit/test_dir/math/client/http"
)

func main() {
	svc, err := http.New("http://localhost:8081", map[string][]http3.ClientOption{})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(svc.Sum(context.Background(), 2, 3))
	fmt.Println(svc.Prod(context.Background(), 2, 3))
}
