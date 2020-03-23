/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"

	"sigs.k8s.io/azuredisk-csi-driver/test/e2e/driver"
	"sigs.k8s.io/azuredisk-csi-driver/test/e2e/testsuites"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

const (
	defaultDiskSize      = 10
	defaultDiskSizeBytes = defaultDiskSize * 1024 * 1024 * 1024
)

var _ = ginkgo.Describe("Pre-Provisioned", func() {
	f := framework.NewDefaultFramework("azuredisk")

	var (
		cs         clientset.Interface
		ns         *v1.Namespace
		testDriver driver.PreProvisionedVolumeTestDriver
		volumeID   string
		// Set to true if the volume should be deleted automatically after test
		skipManuallyDeletingVolume bool
	)

	ginkgo.BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
		testDriver = driver.InitAzureDiskDriver()
	})

	ginkgo.AfterEach(func() {
		if !skipManuallyDeletingVolume {
			req := &csi.DeleteVolumeRequest{
				VolumeId: volumeID,
			}
			_, err := azurediskDriver.DeleteVolume(context.Background(), req)
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("create volume %q error: %v", volumeID, err))
			}
		}
	})

	ginkgo.Context("[single-az]", func() {
		ginkgo.It("should use a pre-provisioned volume and mount it as readOnly in a pod [disk.csi.azure.com]", func() {
			// Az tests need to be changed to pass the right parameters for in-tree driver.
			// Skip these tests until above is fixed.
			if testDriver.IsInTree() {
				ginkgo.Skip("Test running with in tree configuration")
			}

			req := makeCreateVolumeReq("pre-provisioned-readOnly")
			resp, err := azurediskDriver.CreateVolume(context.Background(), req)
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("create volume error: %v", err))
			}
			volumeID = resp.Volume.VolumeId
			ginkgo.By(fmt.Sprintf("Successfully provisioned AzureDisk volume: %q\n", volumeID))

			diskSize := fmt.Sprintf("%dGi", defaultDiskSize)
			pods := []testsuites.PodDetails{
				{
					Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
					Volumes: []testsuites.VolumeDetails{
						{
							VolumeID:  volumeID,
							FSType:    "ext4",
							ClaimSize: diskSize,
							VolumeMount: testsuites.VolumeMountDetails{
								NameGenerate:      "test-volume-",
								MountPathGenerate: "/mnt/test-",
								ReadOnly:          true,
							},
						},
					},
				},
			}
			test := testsuites.PreProvisionedReadOnlyVolumeTest{
				CSIDriver: testDriver,
				Pods:      pods,
			}
			test.Run(cs, ns)
		})

		ginkgo.It(fmt.Sprintf("should use a pre-provisioned volume and retain PV with reclaimPolicy %q [disk.csi.azure.com]", v1.PersistentVolumeReclaimRetain), func() {
			// Az tests need to be changed to pass the right parameters for in-tree driver.
			// Skip these tests until above is fixed.
			if testDriver.IsInTree() {
				ginkgo.Skip("Test running with in tree configuration")
			}
			req := makeCreateVolumeReq("pre-provisioned-retain-reclaimPolicy")
			resp, err := azurediskDriver.CreateVolume(context.Background(), req)
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("create volume error: %v", err))
			}
			volumeID = resp.Volume.VolumeId
			ginkgo.By(fmt.Sprintf("Successfully provisioned AzureDisk volume: %q\n", volumeID))

			diskSize := fmt.Sprintf("%dGi", defaultDiskSize)
			reclaimPolicy := v1.PersistentVolumeReclaimRetain
			volumes := []testsuites.VolumeDetails{
				{
					VolumeID:      volumeID,
					FSType:        "ext4",
					ClaimSize:     diskSize,
					ReclaimPolicy: &reclaimPolicy,
				},
			}
			test := testsuites.PreProvisionedReclaimPolicyTest{
				CSIDriver: testDriver,
				Volumes:   volumes,
			}
			test.Run(cs, ns)
		})

		ginkgo.It("should fail when maxShares is invalid [disk.csi.azure.com][shared disk]", func() {
			// Az tests need to be changed to pass the right parameters for in-tree driver.
			// Skip these tests until above is fixed.
			if testDriver.IsInTree() {
				ginkgo.Skip("Test running with in tree configuration")
			}
			skipManuallyDeletingVolume = true
			req := makeCreateVolumeReq("invalid-maxShares")
			req.Parameters = map[string]string{"maxShares": "0"}
			_, err := azurediskDriver.CreateVolume(context.Background(), req)
			framework.ExpectError(err)
		})
	})
})

func makeCreateVolumeReq(volumeName string) *csi.CreateVolumeRequest {
	req := &csi.CreateVolumeRequest{
		Name: volumeName,
		VolumeCapabilities: []*csi.VolumeCapability{
			{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: defaultDiskSizeBytes,
			LimitBytes:    defaultDiskSizeBytes,
		},
	}

	return req
}
