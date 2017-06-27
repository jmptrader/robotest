package e2e

import (
	"github.com/gravitational/robotest/e2e/runtime"
	"github.com/gravitational/robotest/e2e/uimodel"
	uisite "github.com/gravitational/robotest/e2e/uimodel/site"

	"github.com/gravitational/robotest/e2e/runtime/configs"
	. "github.com/onsi/ginkgo"
)

type clusterCfgType struct {
	ClusterName string `yaml:"cluster_name" validate:"required"`
}

type clusterExpandOnpremCfgType struct {
	Profile string `yaml:"profile" validate:"required"`
}

type clusterExpandAWSCfgType struct {
	Profile      string `yaml:"profile" validate:"required"`
	InstanceType string `yaml:"instance_type" validate:"required"`
}

var _ = RoboDescribe("tests.cluster", func() {
	var clusterURL string
	var clusterCfg = clusterCfgType{}

	tcontext := runtime.New()
	tcontext.BindCfg(&clusterCfg, "tests.cluster")

	BeforeEach(func() {
		clusterURL = tcontext.GetClusterURL(clusterCfg.ClusterName)
	})

	Describe("tests.cluster.expand.onprem", func() {
		var clusterExpandOnpremCfg = clusterExpandOnpremCfgType{}
		tcontext.BindCfg(&clusterExpandOnpremCfg, "tests.cluster.expand.onprem")
		It("should add and remove a server", func() {
			clusterURL := tcontext.GetClusterURL(clusterCfg.ClusterName)
			var cfg = clusterExpandOnpremCfg
			var ui = uimodel.InitWithUser(tcontext.Page, clusterURL, tcontext.GetLogin())

			site := ui.GoToSite(clusterCfg.ClusterName)
			siteServerPage := site.GoToServers()
			newSiteServer := siteServerPage.AddOnPremServer(tcontext, cfg.Profile)
			siteServerPage.DeleteServer(newSiteServer)
		})
	})

	Describe("tests.cluster.expand.aws", func() {
		var clusterExpandAWSCfg = clusterExpandAWSCfgType{}
		tcontext.BindCfg(&clusterExpandAWSCfg, "tests.cluster.expand.aws")

		It("should add and remove a server", func() {
			aws := configs.GetProviderAWS()
			form := uisite.AddAWSServerForm{
				AccessKey:    aws.AccessKey,
				SecretKey:    aws.SecretKey,
				Profile:      clusterExpandAWSCfg.Profile,
				InstanceType: clusterExpandAWSCfg.InstanceType,
			}

			ui := uimodel.InitWithUser(tcontext.Page, clusterURL, tcontext.GetLogin())
			site := ui.GoToSite(clusterCfg.ClusterName)
			siteServerPage := site.GoToServers()
			newServer := siteServerPage.AddAWSServer(form)
			siteServerPage.DeleteServer(newServer)
		})
	})
})
