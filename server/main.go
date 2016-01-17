// Copyright 2016 Google, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	pb "github.com/kelseyhightower/grpc-hello-service/hello"
	healthpb "google.golang.org/grpc/health/grpc_health_v1alpha"

	"github.com/boltdb/bolt"
	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
)

// helloServer is used to implement hello.HelloServer.
type helloServer struct{}

func (hs *helloServer) Say(ctx context.Context, request *pb.Request) (*pb.Response, error) {
	response := &pb.Response{
		Message: fmt.Sprintf("Hello %s", request.Name),
	}

	return response, nil
}

func withConfigDir(path string) string {
	return filepath.Join(os.Getenv("HOME"), ".hello", "server", path)
}

var boltdb *bolt.DB

func main() {
	var (
		caCert          = flag.String("ca-cert", withConfigDir("ca.pem"), "Trusted CA certificate.")
		debugListenAddr = flag.String("debug-listen-addr", "127.0.0.1:8000", "HTTP listen address.")
		listenAddr      = flag.String("listen-addr", "0.0.0.0:443", "HTTP listen address.")
		tlsCert         = flag.String("tls-cert", withConfigDir("cert.pem"), "TLS server certificate.")
		tlsKey          = flag.String("tls-key", withConfigDir("key.pem"), "TLS server key.")
	)
	flag.Parse()

	var err error
	boltdb, err = bolt.Open("hello.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	cert, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
	if err != nil {
		log.Fatal(err)
		return
	}

	rawCaCert, err := ioutil.ReadFile(*caCert)
	if err != nil {
		log.Fatal(err)
		return
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(rawCaCert)

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})

	gs := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterHelloServer(gs, &helloServer{})
	pb.RegisterAuthServer(gs, &loginServer{})

	hs := health.NewHealthServer()
	hs.SetServingStatus("grpc.health.v1.helloservice", 1)
	healthpb.RegisterHealthServer(gs, hs)

	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	go gs.Serve(ln)

	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) { return true, true }
	log.Fatal(http.ListenAndServe(*debugListenAddr, nil))
}