package network // import "github.com/tiborvass/docker/integration/network"

import (
	"runtime"
	"testing"
	"time"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/filters"
	"github.com/tiborvass/docker/api/types/swarm"
	"github.com/tiborvass/docker/client"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/poll"
	"golang.org/x/net/context"
)

func TestServiceWithPredefinedNetwork(t *testing.T) {
	defer setupTest(t)()
	d := newSwarm(t)
	defer d.Stop(t)
	client, err := client.NewClientWithOpts(client.WithHost((d.Sock())))
	assert.NilError(t, err)

	hostName := "host"
	var instances uint64 = 1
	serviceName := "TestService"
	serviceSpec := swarmServiceSpec(serviceName, instances)
	serviceSpec.TaskTemplate.Networks = append(serviceSpec.TaskTemplate.Networks, swarm.NetworkAttachmentConfig{Target: hostName})

	serviceResp, err := client.ServiceCreate(context.Background(), serviceSpec, types.ServiceCreateOptions{
		QueryRegistry: false,
	})
	assert.NilError(t, err)

	pollSettings := func(config *poll.Settings) {
		if runtime.GOARCH == "arm64" || runtime.GOARCH == "arm" {
			config.Timeout = 50 * time.Second
			config.Delay = 100 * time.Millisecond
		} else {
			config.Timeout = 30 * time.Second
			config.Delay = 100 * time.Millisecond
		}
	}

	serviceID := serviceResp.ID
	poll.WaitOn(t, serviceRunningCount(client, serviceID, instances), pollSettings)

	_, _, err = client.ServiceInspectWithRaw(context.Background(), serviceID, types.ServiceInspectOptions{})
	require.NoError(t, err)

	err = client.ServiceRemove(context.Background(), serviceID)
	require.NoError(t, err)

	poll.WaitOn(t, serviceIsRemoved(client, serviceID), pollSettings)
	poll.WaitOn(t, noTasks(client), pollSettings)

}

const ingressNet = "ingress"

func TestServiceWithIngressNetwork(t *testing.T) {
	defer setupTest(t)()
	d := newSwarm(t)
	defer d.Stop(t)

	client, err := client.NewClientWithOpts(client.WithHost((d.Sock())))
	require.NoError(t, err)

	pollSettings := func(config *poll.Settings) {
		if runtime.GOARCH == "arm64" || runtime.GOARCH == "arm" {
			config.Timeout = 50 * time.Second
			config.Delay = 100 * time.Millisecond
		} else {
			config.Timeout = 30 * time.Second
			config.Delay = 100 * time.Millisecond
		}
	}

	poll.WaitOn(t, swarmIngressReady(client), pollSettings)

	var instances uint64 = 1
	serviceName := "TestIngressService"
	serviceSpec := swarmServiceSpec(serviceName, instances)
	serviceSpec.TaskTemplate.Networks = append(serviceSpec.TaskTemplate.Networks, swarm.NetworkAttachmentConfig{Target: ingressNet})
	serviceSpec.EndpointSpec = &swarm.EndpointSpec{
		Ports: []swarm.PortConfig{
			{
				Protocol:    swarm.PortConfigProtocolTCP,
				TargetPort:  80,
				PublishMode: swarm.PortConfigPublishModeIngress,
			},
		},
	}

	serviceResp, err := client.ServiceCreate(context.Background(), serviceSpec, types.ServiceCreateOptions{
		QueryRegistry: false,
	})
	require.NoError(t, err)

	serviceID := serviceResp.ID
	poll.WaitOn(t, serviceRunningCount(client, serviceID, instances), pollSettings)

	_, _, err = client.ServiceInspectWithRaw(context.Background(), serviceID, types.ServiceInspectOptions{})
	assert.NilError(t, err)

	err = client.ServiceRemove(context.Background(), serviceID)
	assert.NilError(t, err)

	poll.WaitOn(t, serviceIsRemoved(client, serviceID), pollSettings)
	poll.WaitOn(t, noTasks(client), pollSettings)

	// Ensure that "ingress" is not removed or corrupted
	time.Sleep(10 * time.Second)
	netInfo, err := client.NetworkInspect(context.Background(), ingressNet, types.NetworkInspectOptions{
		Verbose: true,
		Scope:   "swarm",
	})
	require.NoError(t, err, "Ingress network was removed after removing service!")
	require.NotZero(t, len(netInfo.Containers), "No load balancing endpoints in ingress network")
	require.NotZero(t, len(netInfo.Peers), "No peers (including self) in ingress network")
	_, ok := netInfo.Containers["ingress-sbox"]
	require.True(t, ok, "ingress-sbox not present in ingress network")
}

func serviceRunningCount(client client.ServiceAPIClient, serviceID string, instances uint64) func(log poll.LogT) poll.Result {
	return func(log poll.LogT) poll.Result {
		filter := filters.NewArgs()
		filter.Add("service", serviceID)
		services, err := client.ServiceList(context.Background(), types.ServiceListOptions{})
		if err != nil {
			return poll.Error(err)
		}

		if len(services) != int(instances) {
			return poll.Continue("Service count at %d waiting for %d", len(services), instances)
		}
		return poll.Success()
	}
}

func swarmIngressReady(client client.NetworkAPIClient) func(log poll.LogT) poll.Result {
	return func(log poll.LogT) poll.Result {
		netInfo, err := client.NetworkInspect(context.Background(), ingressNet, types.NetworkInspectOptions{
			Verbose: true,
			Scope:   "swarm",
		})
		if err != nil {
			return poll.Error(err)
		}
		np := len(netInfo.Peers)
		nc := len(netInfo.Containers)
		if np == 0 || nc == 0 {
			return poll.Continue("ingress not ready: %d peers and %d containers", nc, np)
		}
		_, ok := netInfo.Containers["ingress-sbox"]
		if !ok {
			return poll.Continue("ingress not ready: does not contain the ingress-sbox")
		}
		return poll.Success()
	}
}
