package instance

// ListServersResponse represents the response to a ListServers API call
type ListServersResponse struct {
	Servers    Servers
	TotalCount uint32
}
