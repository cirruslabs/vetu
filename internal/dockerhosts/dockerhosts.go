package dockerhosts

import (
	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/types/ref"
	"github.com/samber/lo"
)

func Load(reference ref.Ref, insecure bool) ([]config.Host, error) {
	hosts, err := config.DockerLoad()
	if err != nil {
		return nil, err
	}

	if insecure {
		// Disable TLS for all hosts retrieved from Docker configuration file
		hosts = lo.Map(hosts, func(dockerHost config.Host, index int) config.Host {
			dockerHost.TLS = config.TLSDisabled

			return dockerHost
		})

		// Work around github.com/regclient/regclient not having a WithDefaultTLS(...) option
		// by providing a TLS field override for the registry associated with the passed
		// reference.
		//
		// This means that if the user wants to pull or push from 127.0.0.1:8080/a/b:latest
		// insecurely and Docker configuration contains no such registry, we'll effectively
		// force the regclient to disable TLS for that registry.
		hosts = append(hosts, config.Host{
			Name: reference.Registry,
			TLS:  config.TLSDisabled,
		})
	}

	return hosts, nil
}
