package instance

import (
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/karelorigin/nomad-scaleway-target/types"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// Server is a convenience type for performing operations on a Scaleway server instance
type Server instance.Server

// Decode decodes a map of strings into a server instance
func (s *Server) Decode(config map[string]string) error {
	var shadow struct {
		Name           string            `mapstructure:"name"`
		Tags           types.SliceString `mapstructure:"tags"`
		Zone           string            `mapstructure:"zone"`
		DynamicIP      types.Bool        `mapstructure:"dynamic_ip"`
		CommercialType string            `mapstructure:"commercial_type"`
		Image          *string           `mapstructure:"image"`
		EnableIPv6     types.Bool        `mapstructure:"enable_ipv6"`
		SecurityGroup  *string           `mapstructure:"security_group"`
		PlacementGroup *string           `mapstructure:"placement_group"`
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{DecodeHook: mapstructure.TextUnmarshallerHookFunc(),
		Result: &shadow})
	if err != nil {
		return err
	}

	// Decode into the configuration into the temporary shadow instance
	err = decoder.Decode(config)
	if err != nil {
		return err
	}

	zone, err := scw.ParseZone(shadow.Zone)
	if err != nil {
		return err
	}

	*s = Server(instance.Server{
		Name:              shadow.Name,
		Zone:              zone,
		DynamicIPRequired: bool(shadow.DynamicIP),
		CommercialType:    shadow.CommercialType,
		Tags:              append([]string{"nomad", "client", "autoscaler"}, shadow.Tags...),
		EnableIPv6:        bool(shadow.EnableIPv6),
	})

	// Add instance image if set
	if shadow.Image != nil {
		s.Image = &instance.Image{ID: *shadow.Image}
	}

	// Add security group if set
	if shadow.SecurityGroup != nil {
		s.SecurityGroup = &instance.SecurityGroupSummary{ID: *shadow.SecurityGroup}
	}

	// Add placement group if set
	if shadow.PlacementGroup != nil {
		s.PlacementGroup = &instance.PlacementGroup{ID: *shadow.PlacementGroup}
	}

	return nil
}

// CreateServerRequest creates a Scaleway API request that, when sent, creates a new server
func (s *Server) CreateServerRequest() *instance.CreateServerRequest {
	req := &instance.CreateServerRequest{
		Zone:              s.Zone,
		Name:              s.Name,
		DynamicIPRequired: &s.DynamicIPRequired,
		CommercialType:    s.CommercialType,
		Volumes:           map[string]*instance.VolumeServerTemplate{},
		EnableIPv6:        s.EnableIPv6,
		Tags:              s.Tags,
	}

	// Add image ID if set
	if s.Image != nil {
		req.Image = s.Image.ID
	}

	// Add security group ID if set
	if s.SecurityGroup != nil {
		req.SecurityGroup = &s.SecurityGroup.ID
	}

	// Add placement group ID if set
	if s.PlacementGroup != nil {
		req.PlacementGroup = &s.PlacementGroup.ID
	}

	return req
}

// ListServersRequest creates a Scaleway API request, that when sent, returns a list of server instances
func (s *Server) ListServersRequest() *instance.ListServersRequest {
	var (
		defPage    int32  = 1
		defPerPage uint32 = 100
	)

	req := &instance.ListServersRequest{
		Page:    &defPage,
		PerPage: &defPerPage,
		Tags:    s.Tags,
	}

	// Add name if set
	if len(s.Name) > 0 {
		req.Name = &s.Name
	}

	// Add zone if set
	if len(s.Zone) > 0 {
		req.Zone = s.Zone
	}

	// Add commercial type if set
	if len(s.CommercialType) > 0 {
		req.CommercialType = &s.CommercialType
	}

	return req
}

// ActionAndWaitRequest creates a Scaleway API request that, when sent, changes the state of the server
func (s *Server) ActionAndWaitRequest(action instance.ServerAction, timeout time.Duration) *instance.ServerActionAndWaitRequest {
	return &instance.ServerActionAndWaitRequest{
		ServerID: s.ID,
		Action:   action,
		Timeout:  &timeout,
	}
}

// DeleteServerRequest creates a Scaleway API request that, when sent, removes the server
func (s *Server) DeleteServerRequest() *instance.DeleteServerRequest {
	return &instance.DeleteServerRequest{
		Zone:     s.Zone,
		ServerID: s.ID,
	}
}

// ServerOpt represents a server-related options
type ServerOpt struct {
	UserData types.MapString `mapstructure:"user_data"`
}

// Decode decodes a map of strings into a server options instance
func (s *ServerOpt) Decode(config map[string]string) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{DecodeHook: mapstructure.TextUnmarshallerHookFunc(),
		Result: s})
	if err != nil {
		return err
	}

	err = decoder.Decode(config)
	if err != nil {
		return err
	}

	return nil
}

// Server is a convenience type for performing operations on Scaleway server instances
type Servers []*Server

// NewServers converts a native `[]*instance.Server` type to the `Servers` type
func NewServers(servers []*instance.Server) (s Servers) {
	s = make(Servers, len(servers))

	for i := 0; i < len(s); i++ {
		s[i] = (*Server)(servers[i])
	}

	return s
}

// Ready returns whether all servers are in a running state
func (s Servers) Ready() bool {
	for _, server := range s {
		if server.State != instance.ServerStateRunning {
			return false
		}
	}

	return true
}

// Count returns the amount of servers as a int64
func (s Servers) Count() int64 {
	return int64(len(s))
}

// IDs returns a list of all the server IDs
func (s Servers) IDs() (ids []string) {
	for _, server := range s {
		ids = append(ids, server.ID)
	}

	return ids
}

// WithIDs filters the slice into a subslice of servers with matching IDs
func (s Servers) WithIDs(ids ...string) (r Servers) {
	for _, id := range ids {
		if server := s.WithID(id); server != nil {
			r = append(r, server)
		}
	}

	return r
}

// WithID returns a server by its ID or nil if not found
func (s Servers) WithID(id string) *Server {
	for _, server := range s {
		if server.ID == id {
			return server
		}
	}

	return nil
}

// WithName returns a server by name or nil if not found
func (s Servers) WithName(name string) *Server {
	for _, server := range s {
		if server.Name == name {
			return server
		}
	}

	return nil
}
