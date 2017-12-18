package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/gamelift"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSGameliftAlias_basic(t *testing.T) {
	var conf gamelift.Alias

	rString := acctest.RandString(8)

	aliasName := fmt.Sprintf("tf_acc_alias_%s", rString)
	fleetName := fmt.Sprintf("tf_acc_fleet_%s", rString)
	buildName := fmt.Sprintf("tf_acc_build_%s", rString)
	bucketName := fmt.Sprintf("tf-acc-bucket-gamelift-build-%s", rString)
	roleName := fmt.Sprintf("tf_acc_role_%s", rString)
	policyName := fmt.Sprintf("tf_acc_policy_%s", rString)

	zipPath := "test-fixtures/gamelift-gomoku-build-sample.zip"
	launchPath := `C:\\game\\GomokuServer.exe`

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGameliftAliasDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGameliftAliasBasicConfig(aliasName, fleetName, launchPath, buildName, bucketName, zipPath, roleName, policyName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGameliftAliasExists("aws_gamelift_alias.test", &conf),
					resource.TestCheckResourceAttrSet("aws_gamelift_alias.test", "arn"),
					resource.TestCheckResourceAttr("aws_gamelift_alias.test", "routing_strategy.#", "1"),
					resource.TestCheckResourceAttr("aws_gamelift_alias.test", "name", aliasName),
				),
			},
		},
	})
}

func testAccCheckAWSGameliftAliasExists(n string, res *gamelift.Alias) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Gamelift Alias ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).gameliftconn

		out, err := conn.DescribeAlias(&gamelift.DescribeAliasInput{
			AliasId: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return err
		}
		a := out.Alias

		if *a.AliasId != rs.Primary.ID {
			return fmt.Errorf("Gamelift Alias not found")
		}

		*res = *a

		return nil
	}
}

func testAccCheckAWSGameliftAliasDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).gameliftconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_gamelift_fleet" {
			continue
		}

		_, err := conn.DescribeAlias(&gamelift.DescribeAliasInput{
			AliasId: aws.String(rs.Primary.ID),
		})
		if err == nil {
			return fmt.Errorf("Gamelift Alias still exists")
		}

		return err
	}

	return nil
}

func testAccAWSGameliftAliasBasicConfig(aliasName, fleetName, launchPath, buildName, bucketName, zipPath, roleName, policyName string) string {
	return fmt.Sprintf(`
resource "aws_gamelift_alias" "test" {
  name = "%s"
  routing_strategy {
    message = "test"
    type = "TERMINAL"
  }
}
`, aliasName)
}
