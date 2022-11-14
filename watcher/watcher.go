package watcher

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/bep/debounce"
	"github.com/rs/zerolog/log"
	networking "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

// A Watcher watches for ingresses in the kubernetes cluster
type Watcher struct {
	client    kubernetes.Interface
	context   context.Context
	labels    labels.Selector
	namespace string
	name      string
	//onChange  func(*Payload)
	onChange func([]*networking.Ingress)
}

// New creates a new Watcher.
func New(client kubernetes.Interface, context context.Context, labels labels.Selector, namespace string, name string, onChange func([]*networking.Ingress)) *Watcher {
	return &Watcher{
		client:    client,
		context:   context,
		labels:    labels,
		namespace: namespace,
		name:      name,
		onChange:  onChange,
	}
}

// Run runs the watcher.
func (w *Watcher) Run(ctx context.Context) error {
	factory := informers.NewSharedInformerFactory(w.client, time.Minute)

	serviceLister := factory.Core().V1().Services().Lister()
	ingressLister := factory.Networking().V1().Ingresses().Lister()

	onChange := func() {

		var ingresses []*networking.Ingress
		var err error
		if w.namespace != "" {
			ingresses, err = ingressLister.Ingresses(w.namespace).List(w.labels)
		} else {
			ingresses, err = ingressLister.List(w.labels)
		}

		if err != nil {
			log.Error().Err(err).Msg("failed to list ingresses")
			return
		}

		for _, ingress := range ingresses {

			if w.name != "" && w.name == ingress.Name || w.name == "" {

				hostnames := make([]string, len(ingress.Status.LoadBalancer.Ingress))
				for ix, ig := range ingress.Status.LoadBalancer.Ingress {
					if ig.Hostname != "" {
						hostnames[ix] = ig.Hostname
					} else if ig.IP != "" {
						hostnames[ix] = ig.IP
					}
				}

				dns := strings.Join(hostnames, ",")

				services, errors := serviceLister.Services(ingress.Namespace).List(labels.Everything())

				if errors != nil {
					log.Error().Err(errors).
						Str("namespace", ingress.Namespace).
						Msg("Unable to list services")
				} else {
					for _, service := range services {
						if val, ok := service.Annotations["tickmill.com/nginx.frontrunner"]; ok &&
							service.Spec.Type == "ExternalName" &&
							val == ingress.Name {
							log.Debug().
								Str("service", service.Name).
								Str("type", string(service.Spec.Type)).
								Str("annotation", val).
								Msg("matches")

							// update service
							if service.Spec.ExternalName != dns {
								log.Info().
									Str("service", service.Name).
									Str("old", service.Spec.ExternalName).
									Str("new", dns).
									Msg("Updating")

								servicesClient := w.client.CoreV1().Services(ingress.Namespace)

								retry.RetryOnConflict(retry.DefaultRetry, func() error {
									result, getErr := servicesClient.Get(w.context, service.Name, v1.GetOptions{})
									if getErr != nil {
										log.Error().Err(getErr).Msg("Unable to retrieve service")
										return nil
									}
									result.Spec.ExternalName = dns
									_, updateErr := servicesClient.Update(w.context, result, v1.UpdateOptions{})
									if updateErr == nil {
										log.Info().
											Str("service", service.Name).
											Str("old", service.Spec.ExternalName).
											Str("new", dns).
											Msg("Update completed")
									}
									return updateErr
								})
							}
						}
					}
				}

			}

		}

	}

	debounced := debounce.New(time.Second)
	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			debounced(onChange)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			debounced(onChange)
		},
		DeleteFunc: func(obj interface{}) {
			debounced(onChange)
		},
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		informer := factory.Networking().V1().Ingresses().Informer()
		informer.AddEventHandler(handler)
		informer.Run(ctx.Done())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		// required as without it servicesLister returns empty list
		informer := factory.Core().V1().Services().Informer()
		informer.AddEventHandler(handler)
		informer.Run(ctx.Done())
		wg.Done()
	}()

	wg.Wait()
	return nil
}
