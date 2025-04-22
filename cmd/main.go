package main

import (
	"ec-csi-plugin/pkg/driver"
	"ec-csi-plugin/pkg/version"
	"fmt"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/component-base/cli"
	"log"
	"os"
)

var (
	endpoint     string
	pathToConfig string
	cluster      string
	config       edgecloudV2.CloudConfig
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&pathToConfig, "config", "", "config file")
	if err := rootCmd.MarkPersistentFlagRequired("config"); err != nil {
		log.Fatalf("unable to mark flag config to be required: %v", err)
	}

	rootCmd.PersistentFlags().StringVar(&endpoint, "endpoint", "", "CSI endpoint")
	if err := rootCmd.MarkPersistentFlagRequired("endpoint"); err != nil {
		log.Fatalf("unable to mark flag endpoint to be required: %v", err)
	}

	rootCmd.PersistentFlags().StringVar(&cluster, "cluster", "", "The identifier of the cluster that the plugin is running in.")
	if err := rootCmd.MarkPersistentFlagRequired("cluster"); err != nil {
		log.Fatalf("unable to mark flag cluster to be required: %v", err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "Edgecenter",
	Short: "CSI based Edgecenter driver",
	Run: func(cmd *cobra.Command, args []string) {
		cloud, err := edgecloudV2.NewWithRetries(nil,
			edgecloudV2.SetAPIKey(config.APIToken),
			edgecloudV2.SetBaseURL(config.APIUrl),
			edgecloudV2.SetProject(config.ProjectID),
			edgecloudV2.SetRegion(config.RegionID),
		)
		if err != nil {
			log.Fatalf("failed to create cloud client: %v", err)
			return
		}
		log.Printf("create new driver")
		d := driver.New(cloud, cluster, true)
		d.Start(endpoint)
		d.Wait()
	},
	Version: version.Version,
}

func main() {
	code := cli.Run(rootCmd)
	os.Exit(code)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if pathToConfig != "" {
		// Use config file from the flag.
		viper.SetConfigFile(pathToConfig)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName("ec-csi-plugin.yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("failed to read config file: %s", err)
	}
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("unable to decode the configuration: %s", err)
	}

}
