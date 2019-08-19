package main

import (
	"fmt"
	"github.com/containerd/containerd/api/services/images/v1"
	"github.com/gogo/gateway"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net/http"
)

const (
	restPort   = 8080
	GRPCHeader = "containerd-namespace"
)

var (
	containerdAddr, restAddr string
)

func grpcHandlerFunc(otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Serving REST client")
		otherHandler.ServeHTTP(w, r)
	})
}

func withGRPCNamespaceHeader(ctx context.Context, namespace string) context.Context {
	nsheader := metadata.Pairs(GRPCHeader, namespace)
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = nsheader
	} else {
		md = metadata.Join(nsheader, md)
	}

	return metadata.NewOutgoingContext(ctx, md)
}

func main() {
	containerdAddr = "unix:///run/containerd/containerd.sock"
	restAddr = fmt.Sprintf(":%d", restPort)
	ctx := withGRPCNamespaceHeader(context.Background(), "examplectr")
	dopts := []grpc.DialOption{grpc.WithInsecure()}
	mux := http.NewServeMux()
	m := new(gateway.JSONPb)
	gwmux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, m))
	err := images.RegisterImagesHandlerFromEndpoint(ctx, gwmux, containerdAddr, dopts)
	if err != nil {
		fmt.Printf("Register Images error: %v\n", err)
		return
	}

	mux.Handle("/", gwmux)
	if err != nil {
		panic(err)
	}
	srv := &http.Server{
		Addr:    restAddr,
		Handler: grpcHandlerFunc(mux),
	}

	fmt.Printf("Staring HTTP/REST server on port: %d\n", restPort)
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}

}
