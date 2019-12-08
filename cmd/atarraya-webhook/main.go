package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	whhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	var certFile string
	var keyFile string

	logger := &log.Std{Debug: true}

	logger.Infof("atarraya admission control hook is starting...")

	flag.StringVar(&certFile, "tls-cert-file", "/var/lib/secrets/cert.crt", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&keyFile, "tls-key-file", "/var/lib/secrets/cert.key", "File containing the x509 private key to --tls-cert-file.")

	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		logger.Errorf("Failed to load key pair: %v", err)
	}

	server := webhookServer{
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", 8443),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	mutator := mutating.MutatorFunc(server.atarrayaMutator)

	webhook, err := mutating.NewWebhook(
		mutating.WebhookConfig{Name: "atarraya-mutator", Obj: &corev1.Pod{}},
		mutator,
		nil,
		nil,
		logger)

	handler, err := whhttp.HandlerFor(webhook)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook: %s", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.Handle("/mutate", handler)
	mux.HandleFunc("/health", server.healthHandler)

	server.Server.Handler = mux

	go func() {
		if err := server.Server.ListenAndServeTLS("", ""); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to listen and serve webhook server: %s", err)
			os.Exit(1)
		}
	}()

	logger.Infof("atarraya admission control listening on port 8443")

	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	logger.Infof("Got OS shutdown signal, shutting down webhook server gracefully...")
	server.Server.Shutdown(context.Background())
}
