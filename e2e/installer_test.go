package e2e

import (
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/robotest/e2e/runtime"
	"github.com/gravitational/robotest/e2e/runtime/configs"
	"github.com/gravitational/robotest/e2e/runtime/defaults"
	"github.com/gravitational/robotest/e2e/uimodel"
	"github.com/gravitational/robotest/e2e/uimodel/bandwagon"
	uiinstaller "github.com/gravitational/robotest/e2e/uimodel/installer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type installerCfgType struct {
	ClusterName string                   `yaml:"cluster_name" validate:"required"`
	Bandwagon   *bandwagon.BandwagonForm `yaml:"bandwagon" `
	License     string                   `yaml:"license"`
}

type installerOnpremCfgType struct {
	License        string        `yaml:"license"`
	FlavorLabel    string        `yaml:"flavor_label" validate:"required"`
	DockerDevice   string        `yaml:"docker_device"`
	RemoteAccess   bool          `yaml:"remote_access"`
	InstallTimeout time.Duration `yaml:"install_timeout" `
}

type installerAWSCfgType struct {
	AppURL         string    `yaml:"app_url"`
	FlavorLabel    string    `yaml:"flavor_label"  validate:"required"`
	DockerDevice   string    `yaml:"docker_device"`
	Region         string    `yaml:"region" validate:"required"`
	KeyPair        string    `yaml:"key_pair" validate:"required"`
	VPC            string    `yaml:"vpc"  validate:"required"`
	InstanceType   string    `yaml:"instance_type"  validate:"required"`
	InstallTimeout *duration `yaml:"install_timeout" `
}

var _ = RoboDescribe("tests.installer", func() {
	var installerCfg = installerCfgType{}
	tcontext := runtime.New()
	tcontext.BindCfg(&installerCfg, "tests.installer")

	Describe("tests.installer.onprem", func() {
		var installerOnpremCfg = installerOnpremCfgType{}
		tcontext.BindCfg(&installerOnpremCfg, "tests.installer.onprem")

		It("should provision a new cluster", func() {
			wizardURL := tcontext.GetWizardURL()
			Expect(wizardURL).NotTo(BeEmpty(), "should retrieve wizard URL from provisioner")
			clusterName := installerCfg.ClusterName

			log.Infof("navigating to installer step")
			ui := uimodel.InitWithUser(tcontext.Page, wizardURL, tcontext.GetLogin())
			installer := ui.GoToInstaller(wizardURL)

			log.Infof("filling out license text field if required")
			installer.ProcessLicenseStepIfRequired(installerOnpremCfg.License)

			log.Infof("selecting a provisioner")
			installer.InitOnPremInstallation(clusterName)

			log.Infof("selecting a flavor and allocating the nodes")
			installer.SelectFlavorByLabel(installerOnpremCfg.FlavorLabel)
			installer.PrepareOnPremNodes(tcontext, installerOnpremCfg.DockerDevice)

			log.Infof("starting an installation")
			installer.StartInstallation()

			log.Infof("waiting until install is completed")
			installer.WaitForCompletion(installerOnpremCfg.InstallTimeout)

			log.Infof("checking for bandwagon step")
			if installer.NeedsBandwagon(clusterName) == false {
				ui.GoToSite(clusterName)
				return
			}

			log.Infof("navigating to bandwagon step")
			installer.ProceedToSite()
			bandwagon := ui.GoToBandwagon(clusterName)

			log.Infof("submitting bandwagon form")
			bandwagon.SubmitForm(*installerCfg.Bandwagon)

			log.Infof("navigating to a site and reading endpoints")
			site := ui.GoToSite(clusterName)
			endpoints := site.GetEndpoints()
			endpoints = filterGravityEndpoints(endpoints)
			Expect(len(endpoints)).To(BeNumerically(">", 0), "expected at least one application endpoint")

			log.Infof("login in with bandwagon user credentials")
			siteEntryURL := endpoints[0]
			ui = uimodel.InitWithUser(tcontext.Page, siteEntryURL, tcontext.GetLogin())
			ui.GoToSite(clusterName)
		})
	})

	Describe("tests.installer.aws", func() {

		var installerAWSCfg = installerAWSCfgType{}
		tcontext.BindCfg(&installerAWSCfg, "tests.installer.aws")

		It("should provision a new cluster", func() {
			clusterName := installerCfg.ClusterName
			awsCfg := configs.GetProviderAWS()

			ui := uimodel.InitWithUser(tcontext.Page, installerAWSCfg.AppURL, tcontext.GetLogin())
			installer := ui.GoToInstaller(installerAWSCfg.AppURL)

			log.Infof("filling out license text field if required")
			installer.ProcessLicenseStepIfRequired(installerCfg.License)

			formCfg := uiinstaller.AWSIstallFormCfgType{
				ClusterName: clusterName,
				Region:      installerAWSCfg.Region,
				VPC:         installerAWSCfg.VPC,
				KeyPair:     installerAWSCfg.KeyPair,
				AccessKey:   awsCfg.AccessKey,
				SecretKey:   awsCfg.SecretKey,
			}

			installer.InitAWSInstallation(formCfg)

			log.Infof("selecting a flavor")
			installer.SelectFlavorByLabel(installerAWSCfg.FlavorLabel)
			profiles := installer.GetAWSProfiles()
			Expect(len(profiles)).To(BeNumerically(">", 0), "expect at least 1 profile")

			log.Infof("setting up AWS instance types")
			for _, p := range profiles {
				p.SetInstanceType(installerAWSCfg.InstanceType)
			}

			log.Infof("starting an installation")
			installer.StartInstallation()

			log.Infof("waiting until install is completed or failed")
			installer.WaitForCompletion(installerAWSCfg.InstallTimeout.Duration())

			if installer.NeedsBandwagon(clusterName) {
				log.Infof("navigating to bandwagon step")
				bandwagon := ui.GoToBandwagon(clusterName)
				log.Infof("submitting bandwagon form")

				bandwagon.SubmitForm(*installerCfg.Bandwagon)

				log.Infof("navigating to a site and reading endpoints")
				site := ui.GoToSite(clusterName)
				endpoints := site.GetEndpoints()
				Expect(len(endpoints)).To(BeNumerically(">", 0), "expected at least one application endpoint")
			} else {
				log.Infof("clicking on continue")
				installer.ProceedToSite()
			}
		})
	})

})

func filterGravityEndpoints(endpoints []string) []string {
	var siteEndpoints []string
	for _, v := range endpoints {
		if strings.Contains(v, strconv.Itoa(defaults.GravityHTTPPort)) {
			siteEndpoints = append(siteEndpoints, v)
		}
	}

	return siteEndpoints
}
