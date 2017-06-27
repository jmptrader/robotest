package infra

import (
	"fmt"
	"io/ioutil"

	"github.com/gravitational/log"
	"github.com/gravitational/robotest/e2e/runtime/configs"
	framework "github.com/gravitational/robotest/infra"
	"github.com/gravitational/trace"
)

type InfraName string

const (
	InfraNameLocal  InfraName = "local"
	InfraNameRemote InfraName = "remote"
)

// Infra describes the infrastructure as used in tests.
//
// Infrastructure can be a new cluster that is provisioned as part of the test run
// using one of the built-in provisioners, or an active cluster and Ops Center
// to run tests that require existing infrastructure
type Infra interface {
	GetOpsCenterURL() string
	SetOpsCenterURL(string)
	GetWizardURL() string
	GetDisplayName() string
	GetName() InfraName
	Close() error
	Destroy() error
	Provisioner() framework.Provisioner
	Init() error
}

type InfraState struct {
	DisplayName      string                     `yaml:"name"`
	OpsCenterURL     string                     `yaml:"opscenter_url"`
	WizardURL        string                     `yaml:"wizard_url"`
	ProvisionerName  framework.ProvisionerType  `yaml:"provisioner"`
	ProvisionerState framework.ProvisionerState `yaml:"provisioner_state"`
}

func NewRemote() (Infra, error) {
	var cfg = configs.GetInfraRemote()
	err := configs.Validate(cfg)
	if err != nil {
		return nil, trace.WrapWithMessage(err, "Failed to validate remote infra config section")
	}

	return &infraRemoteType{
		displayName:  cfg.Name,
		opsCenterURL: cfg.URL,
	}, nil
}

func NewRemoteFromState(state InfraState) (Infra, error) {
	return &infraRemoteType{
		displayName:  state.DisplayName,
		opsCenterURL: state.OpsCenterURL,
	}, nil
}

func NewLocal() (Infra, error) {
	var infraCfg = configs.GetInfraLocal()
	err := configs.Validate(infraCfg)
	if err != nil {
		return nil, err
	}

	dirName, err := newStateDir(infraCfg.Name)
	if err != nil {
		return nil, err
	}

	provisioner, err := initProvisioner(
		infraCfg.ProvisionerName,
		infraCfg.Name,
		dirName,
		infraCfg.TarballPath)

	if err != nil {
		return nil, err
	}

	return &localInfra{
		tarballPath: infraCfg.TarballPath,
		name:        infraCfg.Name,
		provisioner: provisioner,
	}, nil
}

func NewLocalFromState(state InfraState) (Infra, error) {
	provisioner, err := initProvisioner(
		configs.ProvisionerName(state.ProvisionerName),
		state.DisplayName,
		"",
		"")

	if err != nil {
		return nil, err
	}

	provisioner.UpdateWithState(state.ProvisionerState)

	return &localInfra{
		name:         state.DisplayName,
		opsCenterURL: state.OpsCenterURL,
		wizardURL:    state.WizardURL,
		provisioner:  provisioner,
	}, nil

}

func GetState(infra Infra) InfraState {
	state := InfraState{
		DisplayName:  infra.GetDisplayName(),
		OpsCenterURL: infra.GetOpsCenterURL(),
		WizardURL:    infra.GetWizardURL(),
	}

	provisioner := infra.Provisioner()
	if provisioner != nil {
		state.ProvisionerName = provisioner.Type()
		state.ProvisionerState = provisioner.State()
	}

	return state
}

func newStateDir(clusterName string) (dir string, err error) {
	dir, err = ioutil.TempDir("", fmt.Sprintf("robotest-%v-", clusterName))
	if err != nil {
		return "", trace.WrapWithMessage(err, "Cannot create state folder")
	}
	log.Infof("state directory: %v", dir)
	return dir, nil
}
