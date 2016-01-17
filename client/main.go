// Copyright 2016 Google, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	pb "github.com/kelseyhightower/grpc-hello-service/hello"
	healthpb "google.golang.org/grpc/health/grpc_health_v1alpha"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func withConfigDir(path string) string {
	return filepath.Join(os.Getenv("HOME"), ".hello", "client", path)
}

func main() {
	var (
		caCert     = flag.String("ca-cert", withConfigDir("ca.pem"), "Trusted CA certificate.")
		serverAddr = flag.String("server-addr", "127.0.0.1:443", "Hello service address.")
		tlsCert    = flag.String("tls-cert", withConfigDir("cert.pem"), "TLS server certificate.")
		tlsKey     = flag.String("tls-key", withConfigDir("key.pem"), "TLS server key.")
	)
	flag.Parse()

	cert, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
	if err != nil {
		log.Fatal(err)
	}

	rawCACert, err := ioutil.ReadFile(*caCert)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(rawCACert)

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	})

	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ac := pb.NewAuthClient(conn)
	lm, err := ac.Login(context.Background(), &pb.LoginRequest{Username: "kelseyhightower", Password: "password"})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(lm.Token)

	c := pb.NewHelloClient(conn)
	message, err := c.Say(context.Background(), &pb.Request{"Kelsey"})
	if err != nil {
		log.Fatal(err)
	}

	log.Println(message.Message)

	log.Println("Starting health check..")
	hc := healthpb.NewHealthClient(conn)
	status, err := hc.Check(context.Background(),
		&healthpb.HealthCheckRequest{Service: "grpc.health.v1.helloservice"})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("status:", status)
}