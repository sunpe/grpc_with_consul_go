package main

import (
	"context"
	"grpc_with_consul_go/client"
	helloworld "grpc_with_consul_go/example/gen-go"
	"log"
	"time"
)

func main() {
	conn, err := client.DialContext(context.Background(),
		"consul://127.0.0.1:8500/DEFAULT_GROUP/helloworld.Greeter")
	if err != nil {
		panic(err)
	}
	defer func() { _ = conn.Close() }()

	greeterClient := helloworld.NewGreeterClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := greeterClient.SayHello(ctx, &helloworld.HelloRequest{Name: "world"})
	if err != nil {
		log.Fatal("greeting error ", err)
	}
	log.Printf("greeting: %s", r.GetMessage())
}
