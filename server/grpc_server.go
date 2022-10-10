package server

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-cleanhttp"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"grpc_with_consul_go/common"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

type GrpcServer struct {
	server   *grpc.Server
	consul   *api.Client
	group    string
	ip       string
	port     int
	services []string
}

func NewGrpcServer(registry string, group string) *GrpcServer {
	server := grpc.NewServer(grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle:     time.Minute,
		MaxConnectionAge:      time.Minute,
		MaxConnectionAgeGrace: time.Second,
		Time:                  time.Second,
		Timeout:               time.Second,
	}))
	grpc_health_v1.RegisterHealthServer(server, health.NewServer())

	consul, err := api.NewClient(&api.Config{Address: registry, Transport: cleanhttp.DefaultPooledTransport()})
	if err != nil {
		panic(fmt.Sprintf("resolve consul path error %v", err))
	}
	if len(group) == 0 {
		group = common.DefaultGroup
	}
	return &GrpcServer{
		server: server,
		consul: consul,
		group:  group,
		ip:     getIpV4(),
	}
}

func (s *GrpcServer) Serve(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	s.port = port
	idleConnClosed := make(chan struct{})

	go func() {
		defer close(idleConnClosed)
		defer s.server.GracefulStop()
		defer s.deregister()

		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, os.Kill)
		<-c
	}()

	//
	s.register()

	log.Printf("## grpc server start at %d ! \n", port)
	if err := s.server.Serve(lis); err != nil {
		panic(err)
	}

	<-idleConnClosed
	log.Println("## grpc server exists! ")
}

func (s *GrpcServer) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.server.RegisterService(desc, impl)
	s.services = append(s.services, desc.ServiceName)
}

func (s *GrpcServer) register() {
	//
	for _, service := range s.services {
		_ = s.consul.Agent().ServiceRegister(&api.AgentServiceRegistration{
			ID:      s.serviceId(service),
			Name:    service,
			Tags:    []string{s.group},
			Port:    s.port,
			Address: s.ip,
			Weights: &api.AgentWeights{Passing: 10, Warning: 1},
			Check: &api.AgentServiceCheck{
				Interval:                       "2s",
				Timeout:                        "6s",
				DeregisterCriticalServiceAfter: "6s",
				GRPC:                           fmt.Sprintf("%s:%d", s.ip, s.port),
			},
		})
	}
}

func (s *GrpcServer) deregister() {
	for _, service := range s.services {
		err := s.consul.Agent().ServiceDeregister(s.serviceId(service))
		log.Printf("## deregister service [%s] error %v", service, err)
	}
}

func (s *GrpcServer) serviceId(service string) string {
	return fmt.Sprintf("%s:%s:%s:%d", s.group, service, s.ip, s.port)
}

func getIpV4() (ip string) {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	for _, address := range addresses {
		// check the address type and if it is not a loop back the display it
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ip = ipNet.IP.String()
				return
			}
		}
	}
	return
}
