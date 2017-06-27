package infra

import framework "github.com/gravitational/robotest/infra"

type infraRemoteType struct {
	opsCenterURL string
	wizardURL    string
	displayName  string
}

func (r *infraRemoteType) GetName() InfraName {
	return InfraNameRemote
}

func (r *infraRemoteType) GetOpsCenterURL() string {
	return r.opsCenterURL
}

func (r *infraRemoteType) SetOpsCenterURL(url string) {
	r.opsCenterURL = url
}

func (r *infraRemoteType) GetWizardURL() string {
	return ""
}

func (r *infraRemoteType) GetDisplayName() string {
	return r.displayName
}

func (r *infraRemoteType) Provisioner() framework.Provisioner {
	return nil
}

func (r *infraRemoteType) Init() error {
	return nil
}

func (r *infraRemoteType) Close() error {
	return nil
}

func (r *infraRemoteType) Destroy() error {
	return nil
}
