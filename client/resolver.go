package client

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-cleanhttp"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
	"grpc_with_consul_go/common"
	"os"
	"os/signal"
	"strings"
	"sync"
)

var logger = grpclog.Component("consul_resolver")

func init() {
	resolver.Register(&consulBuilder{})
}

type consulBuilder struct {
}

func (*consulBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	// target.Url - consul://127.0.0.1:8500/DEFAULT_GROUP/service_name
	consulHost := target.URL.Host
	urlPath := target.URL.Path

	urlPath = strings.Trim(urlPath, "/")
	ps := strings.Split(urlPath, "/")
	group := ps[0]
	serviceName := ps[1]

	consulClient, err := api.NewClient(&api.Config{Address: consulHost, Transport: cleanhttp.DefaultPooledTransport()})
	if err != nil {
		return nil, fmt.Errorf("resolve consul path error %v", err)
	}

	r := &consulResolver{
		client:               consulClient,
		cc:                   cc,
		disableServiceConfig: opts.DisableServiceConfig,
		group:                group,
		serviceName:          serviceName,
	}

	// subscribe and watch
	go r.watch()

	return r, nil
}

func (*consulBuilder) Scheme() string {
	return "consul"
}

type consulResolver struct {
	sync.Mutex
	client               *api.Client
	cc                   resolver.ClientConn
	disableServiceConfig bool
	group                string
	serviceName          string
	lastIndex            uint64
}

func (*consulResolver) ResolveNow(_ resolver.ResolveNowOptions) {

}

func (r *consulResolver) Close() {
}

func (r *consulResolver) watch() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)

	for {
		select {
		case <-c:
			return
		default:
			r.watchService()
		}
	}
}

func (r *consulResolver) watchService() {
	services, meta, err := r.client.Health().Service(r.serviceName, r.group, true,
		&api.QueryOptions{WaitIndex: r.lastIndex})
	if err != nil {
		logger.Errorf("## query health service instance error %v", err)
		return
	}
	r.lastIndex = meta.LastIndex

	addresses := make([]resolver.Address, 0, len(services))
	for _, service := range services {
		addr := resolver.Address{
			Addr:       fmt.Sprintf("%v:%v", service.Service.Address, service.Service.Port),
			ServerName: r.serviceName,
			Attributes: attributes.New(
				common.WeightKey, service.Service.Weights.Passing,
			),
		}
		addresses = append(addresses, addr)
	}

	r.Lock()
	_ = r.cc.UpdateState(resolver.State{Addresses: addresses})
	r.Unlock()
}
