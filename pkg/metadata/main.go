package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	defaultMetadataVersion = "latest"
	metadataURLTemplate    = "http://169.254.169.254/openstack/%s/meta_data.json"
)

func New() Interface {
	return &Metadata{}
}

func (m *Metadata) InstanceID() (string, error) {
	md, err := metaDataInfo()
	if err != nil {
		return "", err
	}
	return md.UUID, nil
}

func (m *Metadata) AZ() (string, error) {
	md, err := metaDataInfo()
	if err != nil {
		return "", err
	}
	return md.AvailabilityZone, nil
}

func metaDataInfo() (*Metadata, error) {
	md, err := request()
	if err != nil {
		return nil, err
	}
	var m Metadata
	err = json.Unmarshal(md, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
func request() ([]byte, error) {
	metadataURL := fmt.Sprintf(metadataURLTemplate, defaultMetadataVersion)
	resp, err := http.Get(metadataURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, err
	}
	return io.ReadAll(resp.Body)
}
