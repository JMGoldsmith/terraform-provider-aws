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

func TestAccAWSGameliftFleet_basic(t *testing.T) {
	var conf gamelift.FleetAttributes

	rString := acctest.RandString(8)

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
		CheckDestroy: testAccCheckAWSGameliftFleetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGameliftFleetBasicConfig(fleetName, launchPath, buildName, bucketName, zipPath, roleName, policyName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGameliftFleetExists("aws_gamelift_fleet.test", &conf),
					resource.TestCheckResourceAttrSet("aws_gamelift_fleet.test", "build_id"),
					resource.TestCheckResourceAttr("aws_gamelift_fleet.test", "ec2_instance_type", "t2.micro"),
					resource.TestCheckResourceAttr("aws_gamelift_fleet.test", "name", fleetName),
				),
			},
		},
	})
}

func testAccCheckAWSGameliftFleetExists(n string, res *gamelift.FleetAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Gamelift Fleet ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).gameliftconn

		out, err := conn.DescribeFleetAttributes(&gamelift.DescribeFleetAttributesInput{
			FleetIds: aws.StringSlice([]string{rs.Primary.ID}),
		})
		if err != nil {
			return err
		}
		attributes := out.FleetAttributes
		if len(attributes) < 1 {
			return fmt.Errorf("Gamelift Fleet %q not found", rs.Primary.ID)
		}
		if len(attributes) != 1 {
			return fmt.Errorf("Expected exactly 1 Gamelift Fleet, found %d under %q",
				len(attributes), rs.Primary.ID)
		}
		fleet := attributes[0]

		if *fleet.FleetId != rs.Primary.ID {
			return fmt.Errorf("Gamelift Fleet not found")
		}

		*res = *fleet

		return nil
	}
}

func testAccCheckAWSGameliftFleetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).gameliftconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_gamelift_fleet" {
			continue
		}

		out, err := conn.DescribeFleetAttributes(&gamelift.DescribeFleetAttributesInput{
			FleetIds: aws.StringSlice([]string{rs.Primary.ID}),
		})
		if err != nil {
			return err
		}

		attributes := out.FleetAttributes

		if len(attributes) > 0 {
			return fmt.Errorf("Gamelift Fleet still exists")
		}

		return nil
	}

	return nil
}

func testAccAWSGameliftFleetBasicConfig(fleetName, launchPath, buildName, bucketName, zipPath, roleName, policyName string) string {
	return fmt.Sprintf(`
resource "aws_gamelift_fleet" "test" {
  build_id = "${aws_gamelift_build.test.id}"
  ec2_instance_type = "t2.micro"
  name = "%s"
  server_launch_path = "%s"
}

resource "aws_gamelift_build" "test" {
  name = "%s"
  operating_system = "WINDOWS_2012"
  storage_location {
    bucket = "${aws_s3_bucket.test.bucket}"
    key = "${aws_s3_bucket_object.test.key}"
    role_arn = "${aws_iam_role.test.arn}"
  }
  depends_on = ["aws_iam_role_policy.test"]
}

resource "aws_s3_bucket" "test" {
  bucket = "%s"
}

resource "aws_s3_bucket_object" "test" {
  bucket = "${aws_s3_bucket.test.bucket}"
  key    = "tf-acc-test-gl-build.zip"
  source = "%s"
  etag   = "${md5(file("%s"))}"
}

resource "aws_iam_role" "test" {
  name = "%s"
  path = "/"
  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "gamelift.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy" "test" {
  name = "%s"
  role = "${aws_iam_role.test.id}"

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "s3:GetObject",
        "s3:GetObjectVersion",
        "s3:GetObjectMetadata"
      ],
      "Resource": "${aws_s3_bucket.test.arn}/*",
      "Effect": "Allow"
    }
  ]
}
POLICY
}
`, fleetName, launchPath, buildName, bucketName, zipPath, zipPath, roleName, policyName)
}
