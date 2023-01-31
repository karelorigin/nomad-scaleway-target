package plugin

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mitchellh/mapstructure"

	"github.com/karelorigin/nomad-scaleway-target/scaleway/instance"
	"github.com/scaleway/scaleway-sdk-go/scw"

	"github.com/hashicorp/nomad-autoscaler/plugins/base"
	"github.com/hashicorp/nomad-autoscaler/plugins/target"
	"github.com/hashicorp/nomad-autoscaler/sdk"
	"github.com/hashicorp/nomad-autoscaler/sdk/helper/nomad"
	"github.com/hashicorp/nomad-autoscaler/sdk/helper/scaleutils"
	"github.com/hashicorp/nomad/api"
)

// Make sure that the plugin satisfies the `target.Target` interface
var _ target.Target = (*Plugin)(nil)

// Plugin represents the Scaleway target plugin
type Plugin struct {
	State
	logger   hclog.Logger
	client   *scw.Client
	instance *instance.API
	cluster  *scaleutils.ClusterScaleUtils
}

// Config represents a plugin configuration object
type Config struct {
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	OrgID     string `mapstructure:"organization_id"`
	ProjectID string `mapstructure:"project_id"`
	Region    string `mapstructure:"region"`
	Zone      string `mapstructure:"zone"`
}

// New returns a new Scaleway target plugin instance
func New(logger hclog.Logger) *Plugin {
	return &Plugin{
		logger: logger,
	}
}

// PluginInfo returns plugin information
func (p *Plugin) PluginInfo() (*base.PluginInfo, error) {
	return &base.PluginInfo{
		Name:       "scaleway",
		PluginType: sdk.PluginTypeTarget,
	}, nil
}

// SetConfig sets the plugin configuration, usually called by the Nomad autoscaler
func (p *Plugin) SetConfig(config map[string]string) error {
	p.logger.Debug("set config", "config", config)

	var conf Config
	err := mapstructure.Decode(config, &conf)
	if err != nil {
		return err
	}

	p.client, err = scw.NewClient(scw.WithAuth(conf.AccessKey, conf.SecretKey),
		scw.WithDefaultProjectID(conf.ProjectID), scw.WithEnv())
	if err != nil {
		return err
	}

	p.instance = instance.NewAPI(p.client)

	p.cluster, err = scaleutils.NewClusterScaleUtils(nomad.ConfigFromNamespacedMap(config), p.logger)
	if err != nil {
		return err
	}

	p.cluster.ClusterNodeIDLookupFunc = p.LookupNodeID

	return nil
}

// Scale performs a scaling action against the target
func (p *Plugin) Scale(action sdk.ScalingAction, config map[string]string) error {
	p.logger.Debug("received scale action", "count", action.Count, "reason", action.Reason)

	p.SetActive()
	defer p.SetIdle()

	// Dry-runs are not supported by Scaleway
	if action.Count == sdk.StrategyActionMetaValueDryRunCount {
		return nil
	}

	var blueprint instance.Server
	err := blueprint.Decode(config)
	if err != nil {
		return err
	}

	var opt instance.ServerOpt
	err = opt.Decode(config)
	if err != nil {
		return err
	}

	servers, err := p.instance.ListServersAll(blueprint)
	if err != nil {
		return err
	}

	p.logger.Debug("scaling", action.Direction, "current servers:", servers.Count())

	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	switch action.Direction {
	case sdk.ScaleDirectionUp:
		return p.ScaleUp(blueprint, action.Count-servers.Count(), &opt)
	case sdk.ScaleDirectionDown:
		return p.ScaleDown(ctx, blueprint, (action.Count-servers.Count())*-1, config)
	case sdk.ScaleDirectionNone:
		return nil
	}

	p.logger.Debug("unknown scale direction:", action.Direction)

	return nil
}

// ScaleUp scales up the server pool by `n` servers, options can be nil
func (p *Plugin) ScaleUp(blueprint instance.Server, n int64, opt *instance.ServerOpt) error {
	num := int(n)
	if num < 0 {
		return fmt.Errorf("n cannot be smaller than 0, got: %d", num)
	}

	ch := make(chan int)
	wg := p.doAsyncScale(num, p.doScaleUp(ch, blueprint, opt))

	// Create n servers
	for i := num; i < num; i++ {
		ch <- i
	}

	close(ch)
	wg.Wait()

	return nil
}

// doScaleUp returns a function that can be used to asynchronously scale up
func (p *Plugin) doScaleUp(ch chan int, blueprint instance.Server, opt *instance.ServerOpt) func() {
	return func() {
		for range ch {
			_, err := p.instance.CreateServer(blueprint, opt)
			if err != nil {
				p.logger.Error("Error while creating scaleway server:", err)
			}
		}
	}
}

// ScaleDown scales down the server pool by `n` servers
func (p *Plugin) ScaleDown(ctx context.Context, blueprint instance.Server, n int64, config map[string]string) error {
	num := int(n)
	if num < 0 {
		return fmt.Errorf("n cannot be smaller than 0, got: %d", n)
	}

	nodes, err := p.ClusterRunPreScaleInTasks(ctx, blueprint, config, num)
	if err != nil {
		return err
	}

	ch := make(chan *instance.Server)
	wg := p.doAsyncScale(len(nodes), p.doScaleDown(ch))

	// Scale down nodes
	for _, node := range nodes {
		ch <- &instance.Server{ID: node.RemoteResourceID, Zone: blueprint.Zone}
	}

	close(ch)
	wg.Wait()

	err = p.cluster.RunPostScaleInTasks(ctx, config, nodes)
	if err != nil {
		return err
	}

	return nil
}

// doScaleUp returns a function that can be used to asynchronously scale down
func (p *Plugin) doScaleDown(ch chan *instance.Server) func() {
	return func() {
		for server := range ch {
			err := p.instance.DeleteServer(server)
			if err != nil {
				p.logger.Error("Error while removing server:", err)
			}
		}
	}
}

// doAsyncScale prepares a number of goroutines and calls the given scaling function
func (p *Plugin) doAsyncScale(count int, fn func()) *sync.WaitGroup {
	threads := int(math.Min(float64(count), 5))

	wg := &sync.WaitGroup{}
	wg.Add(threads)

	for i := 0; i < threads; i++ {
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	return wg
}

// Status fetches information from the Scaleway platform to be used by the Nomad autoscaler
func (p *Plugin) Status(config map[string]string) (*sdk.TargetStatus, error) {
	if p.State.Get() != StateIdle {
		return &sdk.TargetStatus{Ready: false}, nil
	}

	ready, err := p.cluster.IsPoolReady(config)
	if err != nil || !ready {
		return &sdk.TargetStatus{Ready: false}, err
	}

	var blueprint instance.Server
	err = blueprint.Decode(config)
	if err != nil {
		return nil, err
	}

	p.logger.Debug("fetching servers from Scaleway")

	servers, err := p.instance.ListServersAll(blueprint)
	if err != nil {
		return nil, err
	}

	p.logger.Debug("finished fetching servers from Scaleway")

	status := &sdk.TargetStatus{
		Ready: servers.Ready(),
		Count: servers.Count(),
	}

	return status, nil
}

// LookupNodeID translates a Nomad node ID to a Scaleway ID
func (p *Plugin) LookupNodeID(node *api.Node) (id string, err error) {
	name, ok := node.Attributes["unique.hostname"]
	if !ok || len(name) == 0 {
		return id, errors.New("attribute unique.hostname does not exist or has no value")
	}

	blueprint := instance.Server{
		Name: name,
	}

	servers, err := p.instance.ListServersAll(blueprint)
	if err != nil {
		return id, err
	}

	server := servers.WithName(name)
	if server == nil {
		return id, fmt.Errorf("could not find server with hostname '%s'", name)
	}

	return server.ID, err
}

// ClusterRunPreScaleInTasks is a temporary alternative to the built-in `RunPreScaleInTasks`,
// see https://github.com/hashicorp/nomad-autoscaler/issues/572 for more information.
func (p *Plugin) ClusterRunPreScaleInTasks(ctx context.Context, blueprint instance.Server, config map[string]string, num int) ([]scaleutils.NodeResourceID, error) {
	servers, err := p.instance.ListServersAll(blueprint)
	if err != nil {
		return nil, err
	}

	nodes, err := p.cluster.RunPreScaleInTasksWithRemoteCheck(ctx, config, servers.IDs(), num)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}
