package rds_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
)

func TestAccRDSSnapshot_basic(t *testing.T) {
	var v rds.DBSnapshot
	resourceName := "aws_db_snapshot.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, rds.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckDbSnapshotDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshotConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDbSnapshotExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
					acctest.MatchResourceAttrRegionalARN(resourceName, "db_snapshot_arn", "rds", regexp.MustCompile(`snapshot:.+`)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRDSSnapshot_tags(t *testing.T) {
	var v rds.DBSnapshot
	resourceName := "aws_db_snapshot.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, rds.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckDbSnapshotDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshotTags1Config(rName, "key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDbSnapshotExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccSnapshotTags2Config(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDbSnapshotExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccSnapshotTags1Config(rName, "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDbSnapshotExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func TestAccRDSSnapshot_disappears(t *testing.T) {
	var v rds.DBSnapshot
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_db_snapshot.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, rds.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckDbSnapshotDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshotConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDbSnapshotExists(resourceName, &v),
					testAccCheckDbSnapshotDisappears(&v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckDbSnapshotDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).RDSConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_db_snapshot" {
			continue
		}

		request := &rds.DescribeDBSnapshotsInput{
			DBSnapshotIdentifier: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeDBSnapshots(request)

		if tfawserr.ErrCodeEquals(err, rds.ErrCodeDBSnapshotNotFoundFault) {
			continue
		}

		if err == nil {
			for _, dbSnapshot := range resp.DBSnapshots {
				if aws.StringValue(dbSnapshot.DBSnapshotIdentifier) == rs.Primary.ID {
					return fmt.Errorf("AWS DB Snapshot is still exist: %s", rs.Primary.ID)
				}
			}
		}

		return err
	}

	return nil
}

func testAccCheckDbSnapshotExists(n string, v *rds.DBSnapshot) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).RDSConn

		request := &rds.DescribeDBSnapshotsInput{
			DBSnapshotIdentifier: aws.String(rs.Primary.ID),
		}

		response, err := conn.DescribeDBSnapshots(request)
		if err == nil {
			if response.DBSnapshots != nil && len(response.DBSnapshots) > 0 {
				*v = *response.DBSnapshots[0]
				return nil
			}
		}
		return fmt.Errorf("Error finding RDS DB Snapshot %s", rs.Primary.ID)
	}
}

func testAccCheckDbSnapshotDisappears(snapshot *rds.DBSnapshot) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).RDSConn

		if _, err := conn.DeleteDBSnapshot(&rds.DeleteDBSnapshotInput{
			DBSnapshotIdentifier: snapshot.DBSnapshotIdentifier,
		}); err != nil {
			return err
		}

		return nil
	}
}

func testAccSnapshotBaseConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_db_instance" "test" {
  allocated_storage       = 10
  engine                  = "mysql"
  engine_version          = "5.6.35"
  instance_class          = "db.t2.micro"
  name                    = "baz"
  identifier              = %[1]q
  password                = "barbarbarbar"
  username                = "foo"
  maintenance_window      = "Fri:09:00-Fri:09:30"
  backup_retention_period = 0
  parameter_group_name    = "default.mysql5.6"
  skip_final_snapshot     = true
}`, rName)
}

func testAccSnapshotConfig(rName string) string {
	return testAccSnapshotBaseConfig(rName) + fmt.Sprintf(`
resource "aws_db_snapshot" "test" {
  db_instance_identifier = aws_db_instance.test.id
  db_snapshot_identifier = %[1]q
}
`, rName)
}

func testAccSnapshotTags1Config(rName, tag1Key, tag1Value string) string {
	return testAccSnapshotBaseConfig(rName) + fmt.Sprintf(`
resource "aws_db_snapshot" "test" {
  db_instance_identifier = aws_db_instance.test.id
  db_snapshot_identifier = %[1]q

  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tag1Key, tag1Value)
}

func testAccSnapshotTags2Config(rName, tag1Key, tag1Value, tag2Key, tag2Value string) string {
	return testAccSnapshotBaseConfig(rName) + fmt.Sprintf(`
resource "aws_db_snapshot" "test" {
  db_instance_identifier = aws_db_instance.test.id
  db_snapshot_identifier = %[1]q

  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }
}
`, rName, tag1Key, tag1Value, tag2Key, tag2Value)
}
