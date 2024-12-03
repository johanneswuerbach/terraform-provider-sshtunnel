
# Connect to a private database server through a jump server using an SSH tunnel.
#
# An equivalent SSH command would be:
# `ssh -L 15432:db.server:5432 -p 2222 jump@ssh.jump.server -i jump.key`

ephemeral "connection" "internal_db" {
  host        = "ssh.jump.server"
  port        = 2222
  user        = "jump"
  private_key = file("jump.key")

  local_port_forwardings = [{
    local_port  = 15432
    remote_host = "db.server"
    remote_port = 5432
  }]
}
