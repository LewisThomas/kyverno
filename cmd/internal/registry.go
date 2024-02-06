package internal

import (
	"context"
	"errors"

	// "fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/kyverno/kyverno/pkg/config"
	"github.com/kyverno/kyverno/pkg/registryclient"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
)

func setupRegistryClient(ctx context.Context, logger logr.Logger, client kubernetes.Interface) (registryclient.Client, corev1listers.SecretNamespaceLister) {
	logger = logger.WithName("registry-client").WithValues("secrets", imagePullSecrets, "insecure", allowInsecureRegistry)
	logger.Info("setup registry client...")
	factory := kubeinformers.NewSharedInformerFactoryWithOptions(client, resyncPeriod, kubeinformers.WithNamespace(config.KyvernoNamespace()))
	secretLister := factory.Core().V1().Secrets().Lister().Secrets(config.KyvernoNamespace())
	// start informers and wait for cache sync
	if !StartInformersAndWaitForCacheSync(ctx, logger, factory) {
		checkError(logger, errors.New("failed to wait for cache sync"), "failed to wait for cache sync")
	}
	registryOptions := []registryclient.Option{
		registryclient.WithTracing(),
	}
	secrets := strings.Split(imagePullSecrets, ",")
	if imagePullSecrets != "" && len(secrets) > 0 {
		registryOptions = append(registryOptions, registryclient.WithKeychainPullSecrets(secretLister, secrets...))
	}
	if allowInsecureRegistry {
		registryOptions = append(registryOptions, registryclient.WithAllowInsecureRegistry())
	}
	if len(registryCredentialHelpers) > 0 {
		registryOptions = append(registryOptions, registryclient.WithCredentialProviders(strings.Split(registryCredentialHelpers, ",")...))
	}
	registryOptions = append(registryOptions, registryclient.WithClientCertificate("", ""))
	// fmt.Printf("Just added new client certificates via registry.go")

	registryClient, err := registryclient.New(registryOptions...)
	checkError(logger, err, "failed to create registry client")
	return registryClient, secretLister
}
