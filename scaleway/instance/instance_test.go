package instance

import (
	"os"
	"testing"

	"github.com/scaleway/scaleway-sdk-go/scw"
)

// NewTestClient returns a new test client
func NewTestClient() (*scw.Client, error) {
	env := [][]string{
		{"SCW_DEFAULT_REGION", "nl-ams"},
		{"SCW_DEFAULT_ZONE", "nl-ams-1"},
	}

	for _, v := range env {
		os.Setenv(v[0], v[1])
	}

	client, err := scw.NewClient(scw.WithAuth(os.Getenv("SCW_ACCESS_KEY"), os.Getenv("SCW_SECRET_KEY")),
		scw.WithDefaultProjectID(os.Getenv("SCW_DEFAULT_PROJECT_ID")), scw.WithEnv())
	if err != nil {
		return nil, err
	}

	return client, nil
}

// NewTestServer returns a new test server instance
func NewTestServer() (server Server, err error) {
	return server, server.Decode(map[string]string{
		"image":           "bd0565d0-3e72-4ce2-b3b9-9d14df67ec2e",
		"commercial_type": "DEV1-S",
		"dynamic_ip":      "true",
		"enable_ipv6":     "true",
		"zone":            "nl-ams-1", // Why do we need to set this? The env variable should take care of that
		"security_group":  "9aada4ae-7933-43e1-963d-adf066fdeb8b",
	})
}

// NewTestServerOpt returns a new test server options object instance
func NewTestServerOpt() (opt ServerOpt, err error) {
	return opt, opt.Decode(map[string]string{
		"user_data": "foo=bar,hello=world",
	})
}

// TestListServersAll tests the ListServersAll method
func TestListServersAll(t *testing.T) {
	client, err := NewTestClient()
	if err != nil {
		t.Fatal(err)
	}

	server, err := NewTestServer()
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewAPI(client).ListServersAll(server)
	if err != nil {
		t.Fatal(err)
	}
}

// TestCreateServer tests the CreateServer method
func TestCreateServer(t *testing.T) {
	client, err := NewTestClient()
	if err != nil {
		t.Fatal(err)
	}

	server, err := NewTestServer()
	if err != nil {
		t.Fatal(err)
	}

	opt, err := NewTestServerOpt()
	if err != nil {
		t.Fatal(err)
	}

	api := NewAPI(client)

	server, err = api.CreateServer(server, &opt)
	if err != nil {
		t.Fatal(err)
	}

	err = api.DeleteServer(&server)
	if err != nil {
		t.Fatal(err)
	}
}
