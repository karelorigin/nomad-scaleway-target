package instance

import (
	"io"
	"strings"
	"time"

	"github.com/karelorigin/nomad-scaleway-target/types"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// API is a convenience type for interacting with the Scaleway API
type API instance.API

// New returns a new API instance
func NewAPI(client *scw.Client) *API {
	return (*API)(instance.NewAPI(client))
}

// Native returns the original `*instance.API` type
func (a *API) Native() *instance.API {
	return (*instance.API)(a)
}

// ListServers performs the ListServerRequest and returns a list of servers
func (a *API) ListServers(blueprint Server) (*ListServersResponse, error) {
	resp, err := a.Native().ListServers(blueprint.ListServersRequest())
	if err != nil {
		return nil, err
	}

	r := &ListServersResponse{
		Servers:    NewServers(resp.Servers),
		TotalCount: resp.TotalCount,
	}

	return r, nil
}

// ListServersAll iterates over all the pages and returns the sum result
func (a *API) ListServersAll(blueprint Server) (servers Servers, err error) {
	req := blueprint.ListServersRequest()

	for {
		resp, err := a.Native().ListServers(req)
		if err != nil {
			return nil, err
		}

		(*req.Page)++

		// No more servers to fetch, return
		if len(resp.Servers) == 0 {
			return servers, nil
		}

		servers = append(servers, NewServers(resp.Servers)...)
	}
}

// CreateServer creates a new server from the given blueprint
func (a *API) CreateServer(blueprint Server, opt *ServerOpt) (s Server, err error) {
	resp, err := a.Native().CreateServer(blueprint.CreateServerRequest())
	if err != nil {
		return s, err
	}

	var (
		server = Server(*resp.Server)
	)

	err = a.ApplyServerOpt(server, opt)
	if err != nil {
		return s, err
	}

	var (
		timeout = time.Minute * 3
	)

	err = a.Native().ServerActionAndWait(server.ActionAndWaitRequest(instance.ServerActionPoweron, timeout))
	if err != nil {
		return s, err
	}

	return server, nil
}

// ApplyServerOpt applies certain options to a server instance
func (a *API) ApplyServerOpt(server Server, opt *ServerOpt) error {
	if opt == nil {
		return nil
	}

	err := a.ApplyServerUserData(server, opt.UserData)
	if err != nil {
		return err
	}

	return nil
}

// ApplyServerUserData applies the given user data to the given server instance
func (a *API) ApplyServerUserData(server Server, data types.MapString) error {
	if data == nil {
		return nil
	}

	m := make(map[string]io.Reader)
	for k, v := range data {
		m[k] = strings.NewReader(v)
	}

	return a.Native().SetAllServerUserData(&instance.SetAllServerUserDataRequest{
		Zone:     server.Zone,
		ServerID: server.ID,
		UserData: m,
	})
}

// DeleteServer deletes the given server and cleans up any leftover volumes
func (a *API) DeleteServer(server *Server) error {
	var (
		timeout = time.Minute * 5
	)

	err := a.Native().ServerActionAndWait(server.ActionAndWaitRequest(instance.ServerActionPoweroff, timeout))
	if err != nil {
		return err
	}

	err = a.Native().DeleteServer(server.DeleteServerRequest())
	if err != nil {
		return err
	}

	for _, volume := range server.Volumes {
		err := a.Native().DeleteVolume(&instance.DeleteVolumeRequest{Zone: server.Zone, VolumeID: volume.ID})
		if err != nil {
			return err
		}
	}

	return nil
}
