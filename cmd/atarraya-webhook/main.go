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

	"github.com/golang/glog"
	whhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	// set the glog sev to a reasonable default
	flag.Lookup("logtostderr").Value.Set("true")
	// disable logging to disk cause thats strange
	flag.Lookup("log_dir").Value.Set("")
	flag.Lookup("stderrthreshold").Value.Set("INFO")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	var certFile string
	var keyFile string

	glog.Infof("atarraya admission control hook is starting...")

	flag.StringVar(&certFile, "tls-cert-file", "/var/lib/secrets/cert.crt", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&keyFile, "tls-key-file", "/var/lib/secrets/cert.key", "File containing the x509 private key to --tls-cert-file.")

	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		glog.Errorf("Failed to load key pair: %v", err)
	}

	server := webhookServer{
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", 8443),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	mutator := mutating.MutatorFunc(server.atarrayaMutator)
	webhook, err := mutating.NewWebhook(mutating.WebhookConfig{Name: "atarraya-mutator", Obj: &corev1.Pod{}}, mutator, nil, nil, nil)
	handler, err := whhttp.HandlerFor(webhook)
	if err != nil {
		glog.Fatalf("error creating webhook: %s", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/mutate", handler)
	mux.HandleFunc("/health", server.healthHandler)

	server.Server.Handler = mux

	go func() {
		if err := server.Server.ListenAndServeTLS("", ""); err != nil {
			glog.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	glog.Infof("atarraya admission control listening on port 8443")

	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	glog.Infof("Got OS shutdown signal, shutting down webhook server gracefully...")
	server.Server.Shutdown(context.Background())
}
