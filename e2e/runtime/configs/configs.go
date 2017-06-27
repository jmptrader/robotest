package configs

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cloudflare/cfssl/log"
	"github.com/gravitational/robotest/lib/loc"
	"gopkg.in/go-playground/validator.v9"
	yaml "gopkg.in/yaml.v2"

	"github.com/gravitational/trace"
)

var InfraToInit = flag.String("grv-init", "", "infra to initialize")
var FlagTests2Run = flag.String("grv-tests", "", "tests to run")
var StateConfigFile = flag.String("grv-state-file", "config.yaml.state", "State configuration file to use")
var Teardown = flag.Bool("grv-destroy", false, "Destroy infrastructure after all tests")

var configFile = flag.String("grv-config", "config.yaml", "config file in YAML")

type ProvisionerName string

const (
	ProvisionerTerraform ProvisionerName = "terraform"
	ProvisionerVagrant   ProvisionerName = "vagrant"
)

type Locator struct {
	Repository string `json:"repository"`
	Name       string `json:"name"`
	Version    string `json:"version"`
}

type LocatorRef struct {
	*loc.Locator
}

type InfraRemoteConfig struct {
	Name string `yaml:"name" validate:"required"`
	URL  string `yaml:"url" validate:"required"`
}

type InfraLocalConfig struct {
	Name            string          `yaml:"name" validate:"required"`
	TarballPath     string          `yaml:"tarball_path"`
	ProvisionerName ProvisionerName `yaml:"provisioner" validate:"required,eq=terraform|eq=vagrant"`
}

type ProvisionerConfig struct {
	NodeCount  int    `yaml:"nodes" json:"nodes" validate:"required"`
	ScriptPath string `yaml:"script_path" validate:"required"`
	StateDir   string `yaml:"state_dir" json:"state_dir"  `
}

type ProvisionerVagrantConfig struct {
	ProvisionerConfig
}

type ProvisionerTerraformConfig struct {
	ProvisionerConfig
	CloudProvider string `yaml:"provider" validate:"required,eq=aws|eq=azure"`
}

type KeysAWSConfig struct {
	AccessKey string `json:"access_key" yaml:"access_key" validate:"required"`
	SecretKey string `json:"secret_key" yaml:"secret_key" validate:"required"`
}

type ProviderAWSConfig struct {
	KeysAWSConfig
	Region     string `json:"region" yaml:"region" validate:"required"`
	KeyPair    string `json:"key_pair" yaml:"key_pair" validate:"required"`
	SSHKeyPath string `json:"key_path" yaml:"key_path"`
	SSHUser    string `json:"ssh_user" yaml:"ssh_user" validate:"required"`
}

type RuntimeConfig struct {
	Provisioner  ProvisionerName `json:"provisioner" yaml:"provisioner"`
	WebDriverURL string          `json:"web_driver_url,omitempty" yaml:"web_driver_url,omitempty" `
	ReportDir    string          `json:"report_dir" yaml:"report_dir" `
	Teardown     bool            `json:"-" yaml:"-"`
	Login        Login           `json:"login" yaml:"login"`
	ClusterName  string          `json:"cluster_name" yaml:"cluster_name" `
}

// Login defines Ops Center authentication parameters
type Login struct {
	Username     string `json:"username" yaml:"username"  validate:"required"`
	Password     string `json:"password" yaml:"password"`
	AuthProvider string `json:"auth_provider,omitempty" yaml:"auth_provider,omitempty"`
}

var configMap map[interface{}]interface{}

func init() {
	flag.Parse()
	confFile, err := os.Open(*configFile)

	if err != nil && !os.IsNotExist(err) {
		failedToLoad(err)
	}
	if confFile == nil {
		Failf("failed to read configuration from %v", *configFile)
	}

	defer confFile.Close()

	configBytes, err := ioutil.ReadAll(confFile)
	if err != nil {
		failedToLoad(err)
	}

	yaml.Unmarshal(configBytes, &configMap)
	if err != nil {
		failedToLoad(err)
	}

	return
}

func GetRuntime() RuntimeConfig {
	var runtimeCfg = RuntimeConfig{}
	Parse(&runtimeCfg)
	return runtimeCfg
}

func GetInfraRemote() InfraRemoteConfig {
	var cfg = InfraRemoteConfig{}
	Parse(&cfg, "infras", "remote")
	return cfg
}

func GetInfraLocal() InfraLocalConfig {
	var cfg = InfraLocalConfig{}
	Parse(&cfg, "infras", "local")
	return cfg
}

func GetProvisionerByName(provisionerName string) ProvisionerConfig {
	var cfg = ProvisionerConfig{}
	Parse(&cfg, "provisioners", provisionerName)
	return cfg
}

func GetProviderAWS() ProviderAWSConfig {
	var cfg = ProviderAWSConfig{}
	Parse(&cfg, "providers", "aws")
	return cfg
}

func GetTerraformProvisioner() ProvisionerTerraformConfig {
	var baseCfg = ProvisionerConfig{}
	Parse(&baseCfg, "provisioners", "terraform")
	var cfg = ProvisionerTerraformConfig{ProvisionerConfig: baseCfg}
	Parse(&cfg, "provisioners", "terraform")
	return cfg
}

func GetVagrantProvisioner() ProvisionerVagrantConfig {
	var baseCfg = ProvisionerConfig{}
	Parse(&baseCfg, "provisioners", "vagrant")
	var cfg = ProvisionerVagrantConfig{ProvisionerConfig: baseCfg}
	Parse(&cfg, "provisioners", "vagrant")
	return cfg
}

func Parse(object interface{}, path ...interface{}) {
	var pathString = ""
	for _, p := range path {
		pathString = pathString + "." + p.(string)
	}

	tmp := get(configMap, path...)
	out, err := yaml.Marshal(tmp)

	if err != nil {
		failed2Parse(pathString, err)
	}

	err = yaml.Unmarshal(out, object)
	if err != nil {
		failed2Parse(pathString, err)
	}
}

func Validate(obj interface{}) error {
	errors := []error{}
	err := validator.New().Struct(obj)
	if validationErrors, ok := err.(validator.ValidationErrors); err != nil && ok {
		for _, fieldError := range validationErrors {
			errors = append(errors,
				trace.Errorf(` * %s="%v" fails "%s"`, fieldError.Field(), fieldError.Value(), fieldError.Tag()))
		}
	}

	return trace.NewAggregate(errors...)
}

func get(m interface{}, path ...interface{}) interface{} {
	for _, p := range path {
		switch idx := p.(type) {
		case string:
			m = m.(map[interface{}]interface{})[idx]
		case int:
			m = m.([]interface{})[idx]
		}
	}
	return m
}

func set(m interface{}, v interface{}, path ...interface{}) {
	for i, p := range path {
		last := i == len(path)-1
		switch idx := p.(type) {
		case string:
			if last {
				m.(map[interface{}]interface{})[idx] = v
			} else {
				m = m.(map[interface{}]interface{})[idx]
			}
		case int:
			if last {
				m.([]interface{})[idx] = v
			} else {
				m = m.([]interface{})[idx]
			}
		}
	}
}

func Failf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Error(msg)
	panic("Initialization error")
}

func failedToLoad(err error) {
	Failf("failed to read configuration from %v: %v", *configFile, trace.UserMessage(err))
}

var failed2Parse = func(path string, err error) {
	Failf("failed to parse %q section from config file: %v", path, trace.UserMessage(err))
}
