package cmd_test

import (
	. "github.com/cloudfoundry/bosh-micro-cli/cmd"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.google.com/p/gomock/gomock"
	mock_cloud "github.com/cloudfoundry/bosh-micro-cli/cloud/mocks"
	mock_agentclient "github.com/cloudfoundry/bosh-micro-cli/deployer/agentclient/mocks"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	boshsys "github.com/cloudfoundry/bosh-agent/system"
	fakesys "github.com/cloudfoundry/bosh-agent/system/fakes"
	boshuuid "github.com/cloudfoundry/bosh-agent/uuid"

	bmconfig "github.com/cloudfoundry/bosh-micro-cli/config"
	bmdepl "github.com/cloudfoundry/bosh-micro-cli/deployment"
	bmeventlog "github.com/cloudfoundry/bosh-micro-cli/eventlogger"
	bmrel "github.com/cloudfoundry/bosh-micro-cli/release"
	fakeui "github.com/cloudfoundry/bosh-micro-cli/ui/fakes"

	fakebmcpi "github.com/cloudfoundry/bosh-micro-cli/cpi/fakes"
)

var _ = Describe("Cmd/DeleteCmd", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Run", func() {
		var (
			fs                      boshsys.FileSystem
			logger                  boshlog.Logger
			fakeCPIInstaller        *fakebmcpi.FakeInstaller
			uuidGenerator           boshuuid.Generator
			deploymentConfigService bmconfig.DeploymentConfigService
			vmRepo                  bmconfig.VMRepo
			diskRepo                bmconfig.DiskRepo
			stemcellRepo            bmconfig.StemcellRepo
			userConfig              bmconfig.UserConfig

			ui *fakeui.FakeUI

			mockAgentClient        *mock_agentclient.MockAgentClient
			mockAgentClientFactory *mock_agentclient.MockFactory
			mockCloud              *mock_cloud.MockCloud
		)

		var writeDeploymentManifest = func() {
			fs.WriteFileString("/deployment-dir/fake-deployment-manifest.yml", `---
name: test-release

cloud_provider:
  mbus: http://fake-mbus-url
`)
		}

		var writeCPIReleaseTarball = func() {
			fs.WriteFileString("/fake-cpi-release.tgz", "fake-tgz-content")
		}

		var allowCPIToBeExtracted = func() {
			cpiRelease := bmrel.NewRelease(
				"fake-cpi-release-name",
				"fake-cpi-release-version",
				[]bmrel.Job{},
				[]*bmrel.Package{},
				"fake-extracted-dir",
				fs,
			)
			fakeCPIInstaller.SetExtractBehavior("/fake-cpi-release.tgz", cpiRelease, nil)
		}

		var allowCPIToBeInstalled = func() {
			cpiRelease := bmrel.NewRelease(
				"fake-cpi-release-name",
				"fake-cpi-release-version",
				[]bmrel.Job{},
				[]*bmrel.Package{},
				"fake-extracted-dir",
				fs,
			)
			cpiDeployment := bmdepl.CPIDeployment{
				Name: "test-release",
				Mbus: "http://fake-mbus-url",
			}
			fakeCPIInstaller.SetInstallBehavior(cpiDeployment, cpiRelease, mockCloud, nil)
		}

		var newDeleteCmd = func() Cmd {
			deploymentParser := bmdepl.NewParser(fs, logger)
			eventLogger := bmeventlog.NewEventLogger(ui)
			return NewDeleteCmd(
				ui, userConfig, fs, deploymentParser, fakeCPIInstaller,
				vmRepo, diskRepo, stemcellRepo, mockAgentClientFactory,
				eventLogger, logger,
			)
		}

		BeforeEach(func() {
			fs = fakesys.NewFakeFileSystem()
			logger = boshlog.NewLogger(boshlog.LevelNone)
			deploymentConfigService = bmconfig.NewFileSystemDeploymentConfigService("/fake-bosh-deployments.json", fs, logger)
			uuidGenerator = boshuuid.Generator(nil)

			vmRepo = bmconfig.NewVMRepo(deploymentConfigService)
			diskRepo = bmconfig.NewDiskRepo(deploymentConfigService, uuidGenerator)
			stemcellRepo = bmconfig.NewStemcellRepo(deploymentConfigService, uuidGenerator)

			mockCloud = mock_cloud.NewMockCloud(mockCtrl)

			fakeCPIInstaller = fakebmcpi.NewFakeInstaller()

			ui = &fakeui.FakeUI{}

			mockAgentClientFactory = mock_agentclient.NewMockFactory(mockCtrl)
			mockAgentClient = mock_agentclient.NewMockAgentClient(mockCtrl)

			userConfig = bmconfig.UserConfig{
				DeploymentFile: "/deployment-dir/fake-deployment-manifest.yml",
			}

			writeDeploymentManifest()
			writeCPIReleaseTarball()
			allowCPIToBeExtracted()
			allowCPIToBeInstalled()
		})

		Context("when the deployment has not been set", func() {
			BeforeEach(func() {
				userConfig.DeploymentFile = ""
			})

			It("returns an error", func() {
				err := newDeleteCmd().Run([]string{"/fake-cpi-release.tgz"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("No deployment set"))
			})
		})

		Context("when microbosh has been deployed", func() {
			BeforeEach(func() {
				// create deployment manifest yaml file
				deploymentFile := bmconfig.DeploymentFile{
					UUID:              "",
					CurrentVMCID:      "fake-vm-cid",
					CurrentStemcellID: "fake-stemcell-guid",
					CurrentDiskID:     "fake-disk-guid",
					Disks: []bmconfig.DiskRecord{
						{
							ID:   "fake-disk-guid",
							CID:  "fake-disk-cid",
							Size: 100,
						},
					},
					Stemcells: []bmconfig.StemcellRecord{
						{
							ID:  "fake-stemcell-guid",
							CID: "fake-stemcell-cid",
						},
					},
				}
				deploymentConfigService.Save(deploymentFile)
			})

			It("stops the agent, then deletes the vm, disk, and stemcell", func() {
				gomock.InOrder(
					mockAgentClientFactory.EXPECT().Create("http://fake-mbus-url").Return(mockAgentClient),
					mockAgentClient.EXPECT().Stop(),
					mockCloud.EXPECT().DeleteVM("fake-vm-cid"),
					mockCloud.EXPECT().DeleteDisk("fake-disk-cid"),
					mockCloud.EXPECT().DeleteStemcell("fake-stemcell-cid"),
				)

				err := newDeleteCmd().Run([]string{"/fake-cpi-release.tgz"})
				Expect(err).ToNot(HaveOccurred())
			})

			XIt("prints event logging stages", func() {})
		})

		Context("when microbosh has not been deployed", func() {
			BeforeEach(func() {
				deploymentConfigService.Save(bmconfig.DeploymentFile{
					UUID:              "",
					CurrentVMCID:      "",
					CurrentStemcellID: "",
					CurrentDiskID:     "",
				})
			})

			It("returns an error", func() {
				err := newDeleteCmd().Run([]string{"/fake-cpi-release.tgz"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("No existing microbosh instance to delete"))
				Expect(ui.Errors).To(ContainElement("No existing microbosh instance to delete"))

				Expect(len(fakeCPIInstaller.InstallInputs)).To(Equal(0))
			})
		})

		Context("when VM has been deployed", func() {
			BeforeEach(func() {
				deploymentConfigService.Save(bmconfig.DeploymentFile{
					UUID:              "",
					CurrentVMCID:      "fake-vm-cid",
					CurrentStemcellID: "",
					CurrentDiskID:     "",
				})
			})

			It("stops the agent and deletes the VM", func() {
				gomock.InOrder(
					mockAgentClientFactory.EXPECT().Create("http://fake-mbus-url").Return(mockAgentClient),
					mockAgentClient.EXPECT().Stop(),
					mockCloud.EXPECT().DeleteVM("fake-vm-cid"),
				)

				err := newDeleteCmd().Run([]string{"/fake-cpi-release.tgz"})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when a current disk exists", func() {
			BeforeEach(func() {
				deploymentConfigService.Save(bmconfig.DeploymentFile{
					UUID:              "",
					CurrentVMCID:      "",
					CurrentStemcellID: "",
					CurrentDiskID:     "fake-disk-guid",
					Disks: []bmconfig.DiskRecord{
						{
							ID:   "fake-disk-guid",
							CID:  "fake-disk-cid",
							Size: 100,
						},
					},
				})
			})

			It("stops the agent and deletes the VM", func() {
				gomock.InOrder(
					mockAgentClientFactory.EXPECT().Create("http://fake-mbus-url").Return(mockAgentClient),
					mockAgentClient.EXPECT().Stop(),
					mockCloud.EXPECT().DeleteDisk("fake-disk-cid"),
				)

				err := newDeleteCmd().Run([]string{"/fake-cpi-release.tgz"})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when a current stemcell exists", func() {
			BeforeEach(func() {
				deploymentConfigService.Save(bmconfig.DeploymentFile{
					UUID:              "",
					CurrentVMCID:      "",
					CurrentStemcellID: "fake-stemcell-guid",
					CurrentDiskID:     "",
					Stemcells: []bmconfig.StemcellRecord{
						{
							ID:  "fake-stemcell-guid",
							CID: "fake-stemcell-cid",
						},
					},
				})
			})

			It("stops the agent and deletes the VM", func() {
				gomock.InOrder(
					mockAgentClientFactory.EXPECT().Create("http://fake-mbus-url").Return(mockAgentClient),
					mockAgentClient.EXPECT().Stop(),
					mockCloud.EXPECT().DeleteStemcell("fake-stemcell-cid"),
				)

				err := newDeleteCmd().Run([]string{"/fake-cpi-release.tgz"})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})