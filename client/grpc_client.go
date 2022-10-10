package client

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
)

// DialContext create insecure grpc client connection by target and options
// target support `consul://` or default passthrough
func DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	options := make([]grpc.DialOption, 0, len(opts)+4)
	options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials())) // insecure
	options = append(options, grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
	options = append(options, grpc.WithDefaultServiceConfig(
		fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, weightedRoundRobinName)))
	options = append(options, grpc.WithBlock())

	if len(opts) > 0 {
		options = append(options, opts...)
	}

	return grpc.DialContext(ctx, target, options...)
}
