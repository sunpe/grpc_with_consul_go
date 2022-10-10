package client

import (
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"math/rand"
	"sync"
)

// weightedRoundRobinName is the name of weighted round robin balancer.
const weightedRoundRobinName = "my_weighted_round_robin"

// newBuilder creates a new weighted round robin balancer builder.
func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(weightedRoundRobinName, &wrrPickerBuilder{}, base.Config{HealthCheck: true})
}

func init() {
	balancer.Register(newBuilder())
}

type wrrPickerBuilder struct{}

func (*wrrPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	scs := make([]balancer.SubConn, 0, len(info.ReadySCs))
	for subConn, addr := range info.ReadySCs {
		var weight int
		weightValue := addr.Address.Attributes.Value("weight")
		if weightValue != nil {
			weight, _ = weightValue.(int)
		}
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			scs = append(scs, subConn)
		}
	}

	return &wrrPicker{
		subConns: scs,
		// Start at a random index, as the same RR balancer rebuilds a new
		// picker when SubConn states change, and we don't want to apply excess
		// load to the first server in the list.
		next: rand.Intn(len(scs)),
	}
}

type wrrPicker struct {
	// subConns is the snapshot of the roundrobin balancer when this picker was
	// created. The slice is immutable. Each Get() will do a round robin
	// selection from it and return the selected SubConn.
	subConns []balancer.SubConn

	mu   sync.Mutex
	next int
}

func (p *wrrPicker) Pick(balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.Lock()
	sc := p.subConns[p.next]
	p.next = (p.next + 1) % len(p.subConns)
	p.mu.Unlock()
	return balancer.PickResult{SubConn: sc}, nil
}
