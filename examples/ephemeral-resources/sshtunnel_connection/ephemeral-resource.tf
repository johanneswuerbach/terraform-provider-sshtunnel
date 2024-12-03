
# Connect to a private database server through a jump server using an SSH tunnel.
#
# An equivalent SSH command would be:
# `ssh -L 15432:db.server:5432 -p 2222 jump@ssh.jump.server -i jump.key`

ephemeral "sshtunnel_connection" "internal_db" {
  host = "ssh.jump.server"
  port = 2222
  user = "jump"

  auth = {
    private_key = file("jump.key")
  }

  local_port_forwardings = [{
    local_port  = 15432
    remote_host = "db.server"
    remote_port = 5432
  }]
}

# Configure the database provider to connect to the database server through the SSH tunnel.
provider "postgresql" {
  host = "localhost"
  port = ephemeral.sshtunnel_connection.tunnel.local_port_forwardings.0.local_port

  # ...
}
