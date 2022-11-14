package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/calebdoxsey/kubernetes-simple-ingress-controller/watcher"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

var (
	host          string
	port, tlsPort int
	globalClient  *kubernetes.Clientset
	globalCtx     *context.Context
)

func main() {
	flag.StringVar(&host, "host", "0.0.0.0", "the host to bind")
	flag.IntVar(&port, "port", 9080, "the insecure http port")
	flag.IntVar(&tlsPort, "tls-port", 9443, "the secure https port")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	runtime.ErrorHandlers = []func(error){
		func(err error) { log.Warn().Err(err).Msg("[k8s]") },
	}

	client, err := kubernetes.NewForConfig(getKubernetesConfig())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create kubernetes client")
	}

	//s := server.New(server.WithHost(host), server.WithPort(port), server.WithTLSPort(tlsPort))
	w := watcher.New(client, func(payload *watcher.Payload) {
		//s.Update(payload)
		updateServiceTarget(payload)
	})

	eg, ctx := errgroup.WithContext(context.Background())

	globalClient = client
	globalCtx = &ctx

	ingressList, err := client.NetworkingV1().Ingresses("").List(ctx, v1.ListOptions{})

	if err != nil {
		log.Error().Err(err).Msg("failed to list ingresses")
	} else {

		for _, ingress := range ingressList.Items {
			log.Debug().
				Str("namespace", ingress.Namespace).
				Str("name", ingress.Name).
				Msg("As ingress")
		}

	}
	/*
		eg.Go(func() error {
			return s.Run(ctx)
		})
	*/
	eg.Go(func() error {
		return w.Run(ctx)
	})
	if err := eg.Wait(); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func updateServiceTarget(payload *watcher.Payload) {

	for _, ingress := range payload.Ingresses {
		hostnames := make([]string, len(ingress.Ingress.Status.LoadBalancer.Ingress))
		for ix, ig := range ingress.Ingress.Status.LoadBalancer.Ingress {
			if ig.Hostname != "" {
				hostnames[ix] = ig.Hostname
			} else if ig.IP != "" {
				hostnames[ix] = ig.IP
			}
		}

		dns := strings.Join(hostnames, ",")

		log.Info().
			Str("namespace", ingress.Ingress.Namespace).
			Str("name", ingress.Ingress.Name).
			Str("dns", dns).
			Msg("Changed")

		servicesClient := globalClient.CoreV1().Services(ingress.Ingress.Namespace)
		services, err := servicesClient.List(*globalCtx, v1.ListOptions{})
		if err != nil {
			log.Error().Err(err).
				Str("namespace", ingress.Ingress.Namespace).
				Msg("Unable to list services")
		} else {
			for _, service := range services.Items {
				if val, ok := service.Annotations["tickmill.com/nginx.frontrunner"]; ok && service.Spec.Type == "ExternalName" {

					log.Debug().
						Str("service", service.Name).
						Str("type", string(service.Spec.Type)).
						Str("annotation", val).
						Msg("Found")

					if service.Spec.ExternalName != dns {
						log.Info().
							Str("service", service.Name).
							Str("old", service.Spec.ExternalName).
							Str("new", dns).
							Msg("Updating")
						retry.RetryOnConflict(retry.DefaultRetry, func() error {
							result, getErr := servicesClient.Get(*globalCtx, service.Name, v1.GetOptions{})
							if getErr != nil {
								log.Error().Err(getErr).Msg("Unable to retrieve service")
								return nil
							}
							result.Spec.ExternalName = dns
							_, updateErr := servicesClient.Update(*globalCtx, result, v1.UpdateOptions{})
							if updateErr == nil {
								log.Info().
									Str("service", service.Name).
									Str("old", service.Spec.ExternalName).
									Str("new", dns).
									Msg("Updated")
							}
							return updateErr
						})
					}

				}
			}
		}
	}

}

func getKubernetesConfig() *rest.Config {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homeDir(), ".kube", "config"))
	}
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get kubernetes configuration")
	}
	return config
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
