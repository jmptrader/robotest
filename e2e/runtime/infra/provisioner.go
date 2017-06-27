package infra

import (
	"github.com/gravitational/robotest/e2e/runtime/configs"
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/terraform"
	"github.com/gravitational/robotest/infra/vagrant"
	"github.com/gravitational/trace"
)

// This is a temporary adapter between old and new data structures for provision configuration.
// TODO: remote it once old format is gone from the source code
func initProvisioner(provisionerType configs.ProvisionerName, loggerPrefix string, stateDir string, tarballPath string) (provisioner infra.Provisioner, err error) {
	switch provisionerType {
	case configs.ProvisionerTerraform:
		provisioner, err = initTerraform(loggerPrefix, tarballPath, stateDir)
	case configs.ProvisionerVagrant:
		provisioner, err = initVagrant(loggerPrefix, tarballPath, stateDir)
	default:
		err = trace.BadParameter("unknown provisioner %q", provisionerType)
	}

	if err != nil {
		return nil, err
	}

	return provisioner, nil
}

func initTerraform(loggerPrefix string, stateDir string, tarballPath string) (provisioner infra.Provisioner, err error) {
	var cfg = configs.GetTerraformProvisioner()
	if err = configs.Validate(cfg); err != nil {
		return nil, trace.Wrap(err)
	}

	var terraAWS *infra.AWSConfig

	if cfg.CloudProvider == "aws" {
		var aws = configs.GetProviderAWS()
		configs.Validate(aws)
		terraAWS = &infra.AWSConfig{
			AccessKey:  aws.AccessKey,
			SecretKey:  aws.SecretKey,
			Region:     aws.Region,
			KeyPair:    aws.KeyPair,
			SSHKeyPath: aws.SSHKeyPath,
			SSHUser:    aws.SSHUser,
		}
	}

	var terraConfig = &terraform.Config{
		StateDir:      stateDir,
		Config:        infra.Config{ClusterName: loggerPrefix},
		ScriptPath:    cfg.ScriptPath,
		InstallerURL:  tarballPath,
		NumNodes:      cfg.NodeCount,
		OS:            "centos",
		CloudProvider: cfg.CloudProvider,
		AWS:           terraAWS,
	}

	provisioner, err = terraform.New(*terraConfig)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return provisioner, nil
}

func initVagrant(loggerSuffix string, tarballPath string, stateDir string) (provisioner infra.Provisioner, err error) {
	var provCfg = configs.GetVagrantProvisioner()
	if err = configs.Validate(provCfg); err != nil {
		return nil, trace.Wrap(err)
	}

	infraVagrantConfig := vagrant.Config{
		StateDir:     stateDir,
		Config:       infra.Config{ClusterName: loggerSuffix},
		ScriptPath:   provCfg.ScriptPath,
		InstallerURL: tarballPath,
		NumNodes:     provCfg.NodeCount,
	}

	provisioner, err = vagrant.New(infraVagrantConfig)

	if err != nil {
		return nil, trace.Wrap(err)
	}

	return provisioner, nil
}
