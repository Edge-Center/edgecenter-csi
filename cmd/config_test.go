package main

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestDecodeRegionalCloudConfig(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(`apiUrl: https://cloud.invalid
apiToken: fixture
projectID: 341372
regionID: 112
regions:
  dt:
    regionID: 113
`))
	if err != nil {
		t.Fatal(err)
	}
	decoded := config
	if err := v.Unmarshal(&decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.APIUrl != "https://cloud.invalid" || decoded.ProjectID != 341372 ||
		decoded.RegionID != 112 || decoded.Regions["dt"].RegionID != 113 {
		t.Fatal("primary or regional settings were not decoded")
	}
}
