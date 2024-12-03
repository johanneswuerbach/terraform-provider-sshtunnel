package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccEphemeralConnection(t *testing.T) {
	sshHost := readEnv(t, "TEST_SSH_HOST")
	sshPortStr := readEnv(t, "TEST_SSH_PORT")
	sshPort, err := strconv.Atoi(sshPortStr)
	if err != nil {
		t.Fatalf("Error parsing TEST_SSH_PORT: %s", err)
	}
	sshUser := readEnv(t, "TEST_SSH_USER")
	sshPrivateKey := readEnv(t, "TEST_SSH_PRIVATE_KEY")
	remoteHost := readEnv(t, "TEST_REMOTE_HOST")

	config := fmt.Sprintf(`
ephemeral "sshtunnel_connection" "test" {
	host = %[1]q
	port = %[2]d
	user = %[3]q
	private_key = %[4]q
	local_port_forwardings = [{
		local_port = 15432
		remote_host = %[5]q
		remote_port = 5432
	}]
}

provider "echo" {
	data = ephemeral.sshtunnel_connection.test
}

resource "echo" "test" {}
`, sshHost, sshPort, sshUser, sshPrivateKey, remoteHost)

	fmt.Println("Before Test")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			fmt.Println("PreCheck")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						fmt.Println("Inside Check")
						return nil
					},
					resource.TestCheckResourceAttr("echo.test", "data.local_port_forwardings.0.local_port", "15432"),
				),
			},
		},
	})
}

func readEnv(t *testing.T, key string) string {
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Environment variable %q must be set for acceptance tests", key)
	}
	return value
}
