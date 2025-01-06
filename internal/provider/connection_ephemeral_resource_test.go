package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEphemeralConnection(t *testing.T) {
	sshHost := "localhost"
	sshPort := 23333
	key, err := os.ReadFile("../../testing/test-key")
	if err != nil {
		t.Fatalf("Error reading test-key: %s", err)
	}
	sshUser := "terraform"
	sshPrivateKey := string(key)

	remoteHost := "postgresbehindsshtunnel"
	remotePort := 5432

	config := fmt.Sprintf(`
ephemeral "sshtunnel_connection" "test" {
	host = %[1]q
	port = %[2]d
	user = %[3]q

	auth = {
		private_key = %[4]q
	}

	local_port_forwardings = [{
		local_port = 15432
		remote_host = %[5]q
		remote_port = %[6]d
	}]
}

provider "echo" {
	data = ephemeral.sshtunnel_connection.test
}

resource "echo" "test" {}
`, sshHost, sshPort, sshUser, sshPrivateKey, remoteHost, remotePort)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("echo.test", "data.local_port_forwardings.0.local_port", "15432"),
				),
			},
		},
	})
}

func TestAccEphemeralConnection_RandomPort(t *testing.T) {
	sshHost := "localhost"
	sshPort := 23333
	key, err := os.ReadFile("../../testing/test-key")
	if err != nil {
		t.Fatalf("Error reading test-key: %s", err)
	}
	sshUser := "terraform"
	sshPrivateKey := string(key)

	remoteHost := "postgresbehindsshtunnel"
	remotePort := 5432

	config := fmt.Sprintf(`
ephemeral "sshtunnel_connection" "test" {
	host = %[1]q
	port = %[2]d
	user = %[3]q

	auth = {
		private_key = %[4]q
	}

	local_port_forwardings = [{
		remote_host = %[5]q
		remote_port = %[6]d
	}]
}

provider "echo" {
	data = ephemeral.sshtunnel_connection.test
}

resource "echo" "test" {}
`, sshHost, sshPort, sshUser, sshPrivateKey, remoteHost, remotePort)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("echo.test", "data.local_port_forwardings.0.local_port", func(value string) error {
						if value == "" {
							return fmt.Errorf("expected a non-empty string, got %q", value)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccEphemeralConnection_Invalid(t *testing.T) {
	sshHost := "localhost"
	sshPort := 23333
	key, err := os.ReadFile("../../testing/test-key")
	if err != nil {
		t.Fatalf("Error reading test-key: %s", err)
	}
	sshUser := "terraform"
	sshPrivateKey := string(key)

	remoteHost := "postgresbehindsshtunnel"
	remotePort := 15432

	config := fmt.Sprintf(`
ephemeral "sshtunnel_connection" "test" {
	host = %[1]q
	port = %[2]d
	user = %[3]q

	auth = {
		private_key = %[4]q
	}

	local_port_forwardings = [{
		remote_host = %[5]q
		remote_port = %[6]d
	}]
}

provider "echo" {
	data = ephemeral.sshtunnel_connection.test
}

resource "echo" "test" {}
`, sshHost, sshPort, sshUser, sshPrivateKey, remoteHost, remotePort)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("echo.test", "data.local_port_forwardings.0.local_port", func(value string) error {
						if value == "" {
							return fmt.Errorf("expected a non-empty string, got %q", value)
						}
						return nil
					}),
				),
			},
		},
	})
}
