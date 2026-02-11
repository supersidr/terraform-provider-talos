// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package talos_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTalosMachineUpgradeResourceValidation(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccTalosMachineUpgradeResourceConfigInvalidRebootMode(),
				ExpectError: regexp.MustCompile("Attribute reboot_mode value must be one of"),
			},
			{
				Config:   testAccTalosMachineUpgradeResourceConfigValidPlanOnly(),
				PlanOnly: true,
			},
		},
	})
}

func testAccTalosMachineUpgradeResourceConfigInvalidRebootMode() string {
	return `
provider "talos" {}

resource "talos_machine_upgrade" "this" {
  node = "10.5.0.2"

  client_configuration = {
    ca_certificate     = "ca"
    client_certificate = "cert"
    client_key         = "key"
  }

  reboot_mode = "invalid"
}
`
}

func testAccTalosMachineUpgradeResourceConfigValidPlanOnly() string {
	return `
provider "talos" {}

resource "talos_machine_upgrade" "this" {
  node = "10.5.0.2"

  client_configuration = {
    ca_certificate     = "ca"
    client_certificate = "cert"
    client_key         = "key"
  }

  reboot_mode = "default"
  wait        = false
}
`
}
