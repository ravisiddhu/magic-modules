package google

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceSqlDatabaseInstances_basic(t *testing.T) {
	t.Parallel()

	context := map[string]interface{}{
		"random_suffix": randString(t, 10),
	}

	vcrTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccSqlDatabaseInstanceDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceSqlDatabases_basic(context),
				Check: resource.ComposeTestCheckFunc(
					checkDatabasesListDataSourceStateMatchesResourceStateWithIgnores(
						"data.google_sql_databases.qa",
						"google_sql_database.db1",
						"google_sql_database.db2",
						map[string]struct{}{
							"deletion_policy": {},
							"id":              {},
						},
					),
				),
			},
		},
	})
}

func TestAccDataSourceSqlDatabaseInstances_nameFilter(t *testing.T) {
	t.Parallel()

	context := map[string]interface{}{
		"random_suffix": randString(t, 10),
	}

	vcrTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccSqlDatabaseInstanceDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceSqlDatabases_nameFilter(context),
				Check: resource.ComposeTestCheckFunc(
					checkDatabaseListDataSourceStateMatchesResourceStateWithIgnoresForAppliedFilter(
						"data.google_sql_databases.qa",
						"google_sql_database.db2",
						"google_sql_database.db1",
						map[string]struct{}{
							"deletion_policy": {},
							"id":              {},
						},
					),
				),
			},
		},
	})
}

func TestAccDataSourceSqlDatabaseInstances_nameAndCharsetFilter(t *testing.T) {
	t.Parallel()

	context := map[string]interface{}{
		"random_suffix": randString(t, 10),
	}

	vcrTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccSqlDatabaseInstanceDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceSqlDatabases_nameAndCharsetFilter(context),
				Check: resource.ComposeTestCheckFunc(
					checkDatabaseListDataSourceStateMatchesResourceStateWithIgnoresForAppliedFilter(
						"data.google_sql_databases.qa",
						"google_sql_database.db3",
						"google_sql_database.db1",
						map[string]struct{}{
							"deletion_policy": {},
							"id":              {},
						},
					),
				),
			},
		},
	})
}

func testAccDataSourceSqlDatabases_basic(context map[string]interface{}) string {
	return Nprintf(`
resource "google_sql_database_instance" "main" {
  name             = "tf-test-instance-%{random_suffix}"
  database_version = "POSTGRES_14"
  region           = "us-central1"

  settings {
    tier = "db-f1-micro"
  }

  deletion_protection = false
}

resource "google_sql_database" "db1"{
	instance = google_sql_database_instance.main.name
	name = "pg-db1"
}

resource "google_sql_database" "db2"{
	instance = google_sql_database_instance.main.name
	name = "pg-db2"
}

data "google_sql_databases" "qa" {
	instance = google_sql_database_instance.main.name
	depends_on = [
		google_sql_database.db1,
		google_sql_database.db2
	]
}
`, context)
}

func testAccDataSourceSqlDatabases_nameFilter(context map[string]interface{}) string {
	return Nprintf(`
resource "google_sql_database_instance" "main" {
  name             = "tf-test-instance-%{random_suffix}"
  database_version = "MYSQL_8_0"
  region           = "us-central1"

  settings {
    tier = "db-f1-micro"
  }

  deletion_protection = false
}

resource "google_sql_database" "db1"{
	instance = google_sql_database_instance.main.name
	name = "mysql-db1"
}

resource "google_sql_database" "db2"{
	instance = google_sql_database_instance.main.name
	name = "mysql-db2"
}

resource "google_sql_database" "db3"{
	instance = google_sql_database_instance.main.name
	name = "mysql-db3"
}

data "google_sql_databases" "qa" {
	instance = google_sql_database_instance.main.name
	filters{
		name = "name"
		values = [".*[0-9]"]
		exclude_values = [".*2",".*3"]
	}
	depends_on = [
		google_sql_database.db1,
		google_sql_database.db2,
		google_sql_database.db3,
	]
}
`, context)
}

func testAccDataSourceSqlDatabases_nameAndCharsetFilter(context map[string]interface{}) string {
	return Nprintf(`
resource "google_sql_database_instance" "main" {
  name             = "tf-test-instance-%{random_suffix}"
  database_version = "MYSQL_8_0"
  region           = "us-central1"

  settings {
    tier = "db-f1-micro"
  }

  deletion_protection = false
}

resource "google_sql_database" "db1"{
	instance = google_sql_database_instance.main.name
	name = "mysql-db1"
	charset = "UTF8"
}

resource "google_sql_database" "db2"{
	instance = google_sql_database_instance.main.name
	name = "mysql-db2"
	charset = "UTF8"
}

resource "google_sql_database" "db3"{
	instance = google_sql_database_instance.main.name
	name = "mysql-db3"
	charset = "utf8mb4"
  	collation = "utf8mb4_bin"
}

data "google_sql_databases" "qa" {
	instance = google_sql_database_instance.main.name
	filters{
		name = "name"
		values = [".*[0-9]"]
	}
	filters{
		name = "charset"
		values = [".*8"]
		exclude_values = [".*mb4"]
	}
	depends_on = [
		google_sql_database.db1,
		google_sql_database.db2,
		google_sql_database.db3,
	]
}
`, context)
}

// This function checks data source state matches for resorceName database instance state
func checkDatabasesListDataSourceStateMatchesResourceStateWithIgnores(dataSourceName, resourceName, resourceName2 string, ignoreFields map[string]struct{}) func(*terraform.State) error {
	return func(s *terraform.State) error {
		ds, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("can't find %s in state", dataSourceName)
		}

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("can't find %s in state", resourceName)
		}

		rs2, ok := s.RootModule().Resources[resourceName2]
		if !ok {
			return fmt.Errorf("can't find %s in state", resourceName2)
		}

		dsAttr := ds.Primary.Attributes
		rsAttr := rs.Primary.Attributes
		rsAttr2 := rs2.Primary.Attributes

		err := checkDatabaseFieldsMatchForDataSourceStateAndResourceState(dsAttr, rsAttr, ignoreFields)
		if err != nil {
			return err
		}
		err = checkDatabaseFieldsMatchForDataSourceStateAndResourceState(dsAttr, rsAttr2, ignoreFields)
		return err

	}
}

// This function checks state match for resorceName2 and asserts the absense of resorceName in data source
func checkDatabaseListDataSourceStateMatchesResourceStateWithIgnoresForAppliedFilter(dataSourceName, resourceName, resourceName2 string, ignoreFields map[string]struct{}) func(*terraform.State) error {
	return func(s *terraform.State) error {
		ds, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("can't find %s in state", dataSourceName)
		}

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("can't find %s in state", resourceName)
		}

		rs2, ok := s.RootModule().Resources[resourceName2]
		if !ok {
			return fmt.Errorf("can't find %s in state", resourceName2)
		}

		dsAttr := ds.Primary.Attributes
		rsAttr := rs.Primary.Attributes
		rsAttr2 := rs2.Primary.Attributes

		err := checkDatabaseResourceAbsentInDataSourceAfterFilterApllied(dsAttr, rsAttr)
		if err != nil {
			return err
		}
		err = checkDatabaseFieldsMatchForDataSourceStateAndResourceState(dsAttr, rsAttr2, ignoreFields)
		return err

	}
}

// This function asserts the absence of the database instance resource which would not be included in the data source list due to the filter applied.
func checkDatabaseResourceAbsentInDataSourceAfterFilterApllied(dsAttr, rsAttr map[string]string) error {
	totalInstances, err := strconv.Atoi(dsAttr["databases.#"])
	if err != nil {
		return errors.New("Couldn't convert length of instances list to integer")
	}
	for i := 0; i < totalInstances; i++ {
		if dsAttr["databases."+strconv.Itoa(i)+".name"] == rsAttr["name"] {
			return errors.New("The resource is present in data source event after filter applied")
		}
	}
	return nil
}

// This function checks whether all the attributes of the database instance resource and the attributes of the datbase instance inside the data source list are the same
func checkDatabaseFieldsMatchForDataSourceStateAndResourceState(dsAttr, rsAttr map[string]string, ignoreFields map[string]struct{}) error {
	totalInstances, err := strconv.Atoi(dsAttr["databases.#"])
	if err != nil {
		return errors.New("Couldn't convert length of instances list to integer")
	}
	index := "-1"
	for i := 0; i < totalInstances; i++ {
		if dsAttr["databases."+strconv.Itoa(i)+".name"] == rsAttr["name"] {
			index = strconv.Itoa(i)
		}
	}

	if index == "-1" {
		return errors.New("The newly created intance is not found in the data source")
	}

	errMsg := ""
	// Data sources are often derived from resources, so iterate over the resource fields to
	// make sure all fields are accounted for in the data source.
	// If a field exists in the data source but not in the resource, its expected value should
	// be checked separately.
	for k := range rsAttr {
		if _, ok := ignoreFields[k]; ok {
			continue
		}
		if k == "%" {
			continue
		}
		if dsAttr["databases."+index+"."+k] != rsAttr[k] {
			// ignore data sources where an empty list is being compared against a null list.
			if k[len(k)-1:] == "#" && (dsAttr["databases."+index+"."+k] == "" || dsAttr["databases."+index+"."+k] == "0") && (rsAttr[k] == "" || rsAttr[k] == "0") {
				continue
			}
			errMsg += fmt.Sprintf("%s is %s; want %s\n", k, dsAttr["databases."+index+"."+k], rsAttr[k])
		}
	}

	if errMsg != "" {
		return errors.New(errMsg)
	}

	return nil
}
