package main

import (
	"context"
	helloworld "grpc_with_consul_go/example/gen-go"
	"grpc_with_consul_go/server"
	"log"
)

func main() {
	s := server.NewGrpcServer("127.0.0.1:8500", "")
	helloworld.RegisterGreeterServer(s, &HelloService{})
	s.Serve(8000)
}

type HelloService struct {
}

func (s *HelloService) SayHello(_ context.Context, request *helloworld.HelloRequest) (*helloworld.HelloResponse, error) {
	log.Printf("Received: %v", request.GetName())
	return &helloworld.HelloResponse{Message: "Hello " + request.GetName()}, nil
}
