package catalog

import (
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
)

func (sc *MeshCatalog) refreshCache() {
	log.Info().Msg("Refresh cache...")
	servicesCache := make(map[endpoint.WeightedService][]endpoint.Endpoint)
	serviceAccountsCache := make(map[endpoint.NamespacedServiceAccount][]endpoint.NamespacedService)
	// TODO(draychev): split the namespace from the service name -- non-K8s services won't have namespace

	services := sc.meshSpec.ListServices()
	for _, service := range services {
		for _, provider := range sc.endpointsProviders {
			endpoints := provider.ListEndpointsForService(endpoint.ServiceName(service.ServiceName.String()))
			if len(endpoints) == 0 {
				log.Info().Msgf("[%s] No IPs found for service=%s", provider.GetID(), service.ServiceName)
				continue
			}
			log.Trace().Msgf("[%s] Found Endpoints=%s for service=%s", provider.GetID(), endpointsToString(endpoints), service.ServiceName)
			servicesCache[service] = endpoints
		}
	}

	for _, namespacesServiceAccounts := range sc.meshSpec.ListServiceAccounts() {
		for _, provider := range sc.endpointsProviders {
			// TODO (snchh) : remove this provider check once we have figured out the service account story for azure vms
			if provider.GetID() != constants.AzureProviderName {
				log.Trace().Msgf("[%s] Finding Services for servcie acccount =%s", provider.GetID(), namespacesServiceAccounts)
				newServices := provider.ListServicesForServiceAccount(namespacesServiceAccounts)
				if len(newServices) == 0 {
					log.Trace().Msgf("[%s] No services found for service account=%s", provider.GetID(), namespacesServiceAccounts)
					continue
				}
				log.Trace().Msgf("[%s] Found services=%+v for service account=%s", provider.GetID(), newServices, namespacesServiceAccounts)
				if existingServices, exists := serviceAccountsCache[namespacesServiceAccounts]; exists {
					// append only new services i.e. preventing duplication
					for _, service := range newServices {
						isPresent := false
						for _, existingService := range serviceAccountsCache[namespacesServiceAccounts] {
							if existingService.String() == service.String() {
								isPresent = true
							}
							if !isPresent {
								serviceAccountsCache[namespacesServiceAccounts] = append(existingServices, existingService)
							}
						}
					}
				} else {
					serviceAccountsCache[namespacesServiceAccounts] = newServices
				}
			}
		}
	}
	log.Info().Msgf("Services cache: %+v", servicesCache)
	log.Info().Msgf("ServiceAccounts cache: %+v", serviceAccountsCache)
	sc.servicesMutex.Lock()
	sc.servicesCache = servicesCache
	sc.serviceAccountsCache = serviceAccountsCache
	sc.servicesMutex.Unlock()
}
