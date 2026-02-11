resource "talos_machine_upgrade" "this" {
  node = "10.5.0.2"

  client_configuration = {
    ca_certificate     = "base64-encoded-ca"
    client_certificate = "base64-encoded-client-certificate"
    client_key         = "base64-encoded-client-key"
  }

  image       = "ghcr.io/siderolabs/installer:v1.10.0"
  reboot_mode = "default"
  stage       = false
  force       = false
  preserve    = false
  wait        = true
}
