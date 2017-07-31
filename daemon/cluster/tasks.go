package cluster

import (
	swarmapi "github.com/docker/swarmkit/api"
	apitypes "github.com/moby/moby-core/api/types"
	"github.com/moby/moby-core/api/types/filters"
	types "github.com/moby/moby-core/api/types/swarm"
	"github.com/moby/moby-core/daemon/cluster/convert"
	"golang.org/x/net/context"
)

// GetTasks returns a list of tasks matching the filter options.
func (c *Cluster) GetTasks(options apitypes.TaskListOptions) ([]types.Task, error) {
	var r *swarmapi.ListTasksResponse

	if err := c.lockedManagerAction(func(ctx context.Context, state nodeState) error {
		filterTransform := func(filter filters.Args) error {
			if filter.Include("service") {
				serviceFilters := filter.Get("service")
				for _, serviceFilter := range serviceFilters {
					service, err := getService(ctx, state.controlClient, serviceFilter, false)
					if err != nil {
						return err
					}
					filter.Del("service", serviceFilter)
					filter.Add("service", service.ID)
				}
			}
			if filter.Include("node") {
				nodeFilters := filter.Get("node")
				for _, nodeFilter := range nodeFilters {
					node, err := getNode(ctx, state.controlClient, nodeFilter)
					if err != nil {
						return err
					}
					filter.Del("node", nodeFilter)
					filter.Add("node", node.ID)
				}
			}
			if !filter.Include("runtime") {
				// default to only showing container tasks
				filter.Add("runtime", "container")
				filter.Add("runtime", "")
			}
			return nil
		}

		filters, err := newListTasksFilters(options.Filters, filterTransform)
		if err != nil {
			return err
		}

		r, err = state.controlClient.ListTasks(
			ctx,
			&swarmapi.ListTasksRequest{Filters: filters})
		return err
	}); err != nil {
		return nil, err
	}

	tasks := make([]types.Task, 0, len(r.Tasks))
	for _, task := range r.Tasks {
		t, err := convert.TaskFromGRPC(*task)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// GetTask returns a task by an ID.
func (c *Cluster) GetTask(input string) (types.Task, error) {
	var task *swarmapi.Task
	if err := c.lockedManagerAction(func(ctx context.Context, state nodeState) error {
		t, err := getTask(ctx, state.controlClient, input)
		if err != nil {
			return err
		}
		task = t
		return nil
	}); err != nil {
		return types.Task{}, err
	}
	return convert.TaskFromGRPC(*task)
}
