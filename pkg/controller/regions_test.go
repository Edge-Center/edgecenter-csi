package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"ec-csi-plugin/pkg/region"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const volumeUUID = "7b9cf879-7146-417c-abfd-cb4272f0c935"

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func fakeScope(t *testing.T, name string, id int, serve func(*http.Request) (int, string)) region.Scope {
	t.Helper()
	httpClient := &http.Client{Transport: roundTripper(func(req *http.Request) (*http.Response, error) {
		if !strings.HasPrefix(req.URL.Path, fmt.Sprintf("/v1/volumes/7/%d", id)) {
			t.Fatalf("unexpected API path: %s", req.URL.Path)
		}
		code, body := serve(req)
		return &http.Response{
			StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(body)), Request: req,
		}, nil
	})}
	client, err := edgecloud.New(httpClient, edgecloud.SetBaseURL("https://cloud.invalid/"), edgecloud.SetAPIKey("test"),
		edgecloud.SetProject(7), edgecloud.SetRegion(id))
	if err != nil {
		t.Fatal(err)
	}
	return region.Scope{Name: name, Config: region.Config{ProjectID: 7, RegionID: id}, Client: client}
}

func TestCreateVolumeRegionalRetry(t *testing.T) {
	dm := fakeScope(t, "default", 112, func(req *http.Request) (int, string) {
		if req.Method != http.MethodGet {
			t.Fatal("retry must not create a second disk in DM")
		}
		return http.StatusOK, `{"results":[]}`
	})
	dt := fakeScope(t, "dt", 113, func(req *http.Request) (int, string) {
		if req.Method != http.MethodGet {
			t.Fatal("retry must reuse the DT disk")
		}
		return http.StatusOK, `{"results":[{"id":"` + volumeUUID + `","name":"pvc-test","size":16}]}`
	})
	driver := New(&region.Set{Scopes: []region.Scope{dm, dt}}, "test", logrus.NewEntry(logrus.New()))
	req := &csi.CreateVolumeRequest{
		Name: "pvc-test",
		VolumeCapabilities: []*csi.VolumeCapability{{
			AccessMode: supportedAccessMode,
			AccessType: &csi.VolumeCapability_Block{Block: &csi.VolumeCapability_BlockVolume{}},
		}},
		AccessibilityRequirements: &csi.TopologyRequirement{
			Preferred: []*csi.Topology{dt.Topology(), dm.Topology()},
			Requisite: []*csi.Topology{dm.Topology(), dt.Topology()},
		},
	}
	created, err := driver.CreateVolume(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	volume := created.GetVolume()
	topologies := volume.GetAccessibleTopology()
	if volume.GetVolumeId() != dt.Handle(volumeUUID) || len(topologies) != 1 ||
		topologies[0].GetSegments()[region.TopologyKey] != "7-113" {
		t.Fatalf("unexpected handle or accessible topology: %v", volume)
	}
	req.AccessibilityRequirements.Preferred = []*csi.Topology{dm.Topology(), dt.Topology()}
	if _, err := driver.CreateVolume(context.Background(), req); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected rejection on retry in the wrong region, got %v", err)
	}
}

func TestDeleteRoutesLegacyAndRegionalHandles(t *testing.T) {
	var called []int
	scope := func(name string, id int) region.Scope {
		return fakeScope(t, name, id, func(req *http.Request) (int, string) {
			if req.Method != http.MethodGet || !strings.HasSuffix(req.URL.Path, "/"+volumeUUID) {
				t.Fatalf("unexpected request for missing disk: %s %s", req.Method, req.URL.Path)
			}
			called = append(called, id)
			return http.StatusNotFound, `{"message":"missing volume"}`
		})
	}
	dm, dt := scope("default", 112), scope("dt", 113)
	driver := New(&region.Set{Scopes: []region.Scope{dm, dt}}, "test", logrus.NewEntry(logrus.New()))
	for _, handle := range []string{volumeUUID, dt.Handle(volumeUUID)} {
		if _, err := driver.DeleteVolume(context.Background(), &csi.DeleteVolumeRequest{VolumeId: handle}); err != nil {
			t.Fatal(err)
		}
	}
	if fmt.Sprint(called) != "[112 113]" {
		t.Fatalf("unexpected API scopes: %v", called)
	}
	if _, err := driver.DeleteVolume(context.Background(), &csi.DeleteVolumeRequest{VolumeId: "ec:7:999:" + volumeUUID}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("must reject an unknown scope instead of deleting elsewhere: %v", err)
	}
}
