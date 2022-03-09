package scaleway

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/scaleway/scaleway-sdk-go/api/baremetal/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_baremetal_server", &resource.Sweeper{
		Name: "scaleway_baremetal_server",
		F:    testSweepBaremetalServer,
	})
}

func testSweepBaremetalServer(_ string) error {
	return sweepZones([]scw.Zone{scw.ZoneFrPar2}, func(scwClient *scw.Client, zone scw.Zone) error {
		baremetalAPI := baremetal.NewAPI(scwClient)
		l.Debugf("sweeper: destroying the baremetal server in (%s)", zone)
		listServers, err := baremetalAPI.ListServers(&baremetal.ListServersRequest{Zone: zone}, scw.WithAllPages())
		if err != nil {
			l.Warningf("error listing servers in (%s) in sweeper: %s", zone, err)
			return nil
		}

		for _, server := range listServers.Servers {
			_, err := baremetalAPI.DeleteServer(&baremetal.DeleteServerRequest{
				Zone:     zone,
				ServerID: server.ID,
			})
			if err != nil {
				return fmt.Errorf("error deleting server in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewayBaremetalServer_Basic(t *testing.T) {
	//t.Skip("Skipping Baremetal Server test as no stock is available currently")
	tt := NewTestTools(t)
	defer tt.Cleanup()

	SSHKeyName := "TestAccScalewayBaremetalServer_Basic"
	SSHKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIM7HUxRyQtB2rnlhQUcbDGCZcTJg7OvoznOiyC9W6IxH opensource@scaleway.com"
	name := "TestAccScalewayBaremetalServer_Basic"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayBaremetalServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_account_ssh_key" "main" {
						name 	   = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
						name        = "%s"
						zone        = "fr-par-2"
						description = "test a description"
						offer       = "EM-L101X-SATA"
						os          = "e04414b9-16d4-4ffb-8ba1-6b80704c7d2a"
					
						tags = [ "terraform-test", "scaleway_baremetal_server", "minimal" ]
						ssh_key_ids = [ scaleway_account_ssh_key.main.id ]
					}
				`, SSHKeyName, SSHKey, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "name", name),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "offer_id", "fr-par-2/bd757ca3-a71b-4158-9ef5-39436b6db2a"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "os_id", "fr-par-2/e04414b9-16d4-4ffb-8ba1-6b80704c7d2a"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "description", "test a description"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.0", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.1", "scaleway_baremetal_server"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.2", "minimal"),
					testCheckResourceAttrUUID("scaleway_baremetal_server.base", "ssh_key_ids.0"),
				),
			},
			{
				// Trigger a reinstall and update tags
				Config: fmt.Sprintf(`
					resource "scaleway_account_ssh_key" "main" {
						name 	   = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
						name        = "%s"
						zone        = "fr-par-2"
						description = "test a description"
						offer       = "EM-L101X-SATA"
						os          = "e04414b9-16d4-4ffb-8ba1-6b80704c7d2a"
					
						tags = [ "terraform-test", "scaleway_baremetal_server", "minimal", "edited" ]
						ssh_key_ids = [ scaleway_account_ssh_key.main.id ]
					}
				`, SSHKeyName, SSHKey, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "name", name),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "offer_id", "fr-par-2/bd757ca3-a71b-4158-9ef5-39436b6db2a4"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "os_id", "fr-par-2/e04414b9-16d4-4ffb-8ba1-6b80704c7d2a"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "description", "test a description"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.#", "4"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.0", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.1", "scaleway_baremetal_server"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.2", "minimal"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.3", "edited"),
					testCheckResourceAttrUUID("scaleway_baremetal_server.base", "ssh_key_ids.0"),
				),
			},
		},
	})
}

func testAccCheckScalewayBaremetalServerExists(tt *TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		baremetalAPI, zonedID, err := baremetalAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = baremetalAPI.GetServer(&baremetal.GetServerRequest{
			ServerID: zonedID.ID,
			Zone:     zonedID.Zone,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayBaremetalServerDestroy(tt *TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_baremetal_server" {
				continue
			}

			baremetalAPI, zonedID, err := baremetalAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = baremetalAPI.GetServer(&baremetal.GetServerRequest{
				ServerID: zonedID.ID,
				Zone:     zonedID.Zone,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("server (%s) still exists", rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !is404Error(err) {
				return err
			}
		}
		return nil
	}
}
