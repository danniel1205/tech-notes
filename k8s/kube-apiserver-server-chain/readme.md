# kube-apiserver-server-chain

## Entry point

```go
// Run runs the specified APIServer.  This should never exit.
func Run(completeOptions completedServerRunOptions, stopCh <-chan struct{}) error {
	// To help debugging, immediately log version
	klog.Infof("Version: %+v", version.Get())

	klog.InfoS("Golang settings", "GOGC", os.Getenv("GOGC"), "GOMAXPROCS", os.Getenv("GOMAXPROCS"), "GOTRACEBACK", os.Getenv("GOTRACEBACK"))

	server, err := CreateServerChain(completeOptions, stopCh)
	if err != nil {
		return err
	}

	prepared, err := server.PrepareRun()
	if err != nil {
		return err
	}

	return prepared.Run(stopCh)
}
```

- CreateServerChain
- PrepareRun
- Run

## Server Chain

```go
// k8s.io/kubernetes/cmd/kube-apiserver/app/server.go

// CreateServerChain creates the apiservers connected via delegation.
func CreateServerChain(completedOptions completedServerRunOptions, stopCh <-chan struct{}) (*aggregatorapiserver.APIAggregator, error) {
kubeAPIServerConfig, serviceResolver, pluginInitializer, err := CreateKubeAPIServerConfig(completedOptions)
if err != nil {
return nil, err
}

// If additional API servers are added, they should be gated.
apiExtensionsConfig, err := createAPIExtensionsConfig(*kubeAPIServerConfig.GenericConfig, kubeAPIServerConfig.ExtraConfig.VersionedInformers,
	pluginInitializer, completedOptions.ServerRunOptions, completedOptions.MasterCount,
serviceResolver, webhook.NewDefaultAuthenticationInfoResolverWrapper(kubeAPIServerConfig.ExtraConfig.ProxyTransport,
	kubeAPIServerConfig.GenericConfig.EgressSelector, kubeAPIServerConfig.GenericConfig.LoopbackClientConfig,
	kubeAPIServerConfig.GenericConfig.TracerProvider))
if err != nil {
return nil, err
}

notFoundHandler := notfoundhandler.New(kubeAPIServerConfig.GenericConfig.Serializer, genericapifilters.NoMuxAndDiscoveryIncompleteKey)
apiExtensionsServer, err := createAPIExtensionsServer(apiExtensionsConfig, genericapiserver.NewEmptyDelegateWithCustomHandler(notFoundHandler))
if err != nil {
return nil, err
}

kubeAPIServer, err := CreateKubeAPIServer(kubeAPIServerConfig, apiExtensionsServer.GenericAPIServer)
if err != nil {
return nil, err
}

// aggregator comes last in the chain
aggregatorConfig, err := createAggregatorConfig(*kubeAPIServerConfig.GenericConfig, completedOptions.ServerRunOptions,
	kubeAPIServerConfig.ExtraConfig.VersionedInformers, serviceResolver, kubeAPIServerConfig.ExtraConfig.ProxyTransport, pluginInitializer)
if err != nil {
return nil, err
}
aggregatorServer, err := createAggregatorServer(aggregatorConfig, kubeAPIServer.GenericAPIServer, apiExtensionsServer.Informers)
if err != nil {
// we don't need special handling for innerStopCh because the aggregator server doesn't create any go routines
return nil, err
}

return aggregatorServer, nil
}

```

- createAPIExtensionsServer
- CreateKubeAPIServer
- createAggregatorServer

Above three steps build up the server chain.

## References

- [Differences between AA and CRD](https://github.com/kubernetes-sigs/apiserver-builder-alpha/blob/master/docs/compare_with_kubebuilder.md)
- [Client-go SharedInformer(Chinese Version)](https://codeantenna.com/a/AwnDgzqPFa)
- [Deep dive with code walk through on kube-apiserver(Chinese Version)](https://blog.tianfeiyu.com/source-code-reading-notes/kubernetes/kube_apiserver.html)
