package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	"github.com/Tickmill-Ltd/service-name-from-ingress/watcher"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	ingressLabels, ingressNamespace, ingressName, logLevel string
)

func main() {
	flag.StringVar(&ingressLabels, "ingress.labels", "", "Watch for ingress with this label")
	flag.StringVar(&ingressNamespace, "ingress.namespace", "", "Watch for ingress in this namespace")
	flag.StringVar(&ingressName, "ingress.name", "", "Watch for ingress with this name")
	flag.StringVar(&logLevel, "log.level", "info", "log level (debug|info|warning|error)")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Fatal().Err(err).Send()

	}
	zerolog.SetGlobalLevel(level)

	runtime.ErrorHandlers = []func(error){
		func(err error) { log.Warn().Err(err).Msg("[k8s]") },
	}

	log.Info().
		Str("ingress.lables", ingressLabels).
		Str("ingress.namespace", ingressNamespace).
		Str("ingress.name", ingressName).
		Msg("Started")

	client, err := kubernetes.NewForConfig(getKubernetesConfig())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create kubernetes client")
	}

	eg, ctx := errgroup.WithContext(context.Background())

	w := watcher.New(client, ctx, labels.Everything(), ingressNamespace, ingressName, func(ingresses []*networking.Ingress) {

		return
	})

	eg.Go(func() error {
		return w.Run(ctx)
	})
	if err := eg.Wait(); err != nil {
		log.Fatal().Err(err).Send()
	} else {
		log.Info().Msg("Stopping")
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
