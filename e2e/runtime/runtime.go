package runtime

import (
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/robotest/e2e/runtime/configs"
	runtimeinfra "github.com/gravitational/robotest/e2e/runtime/infra"
	"github.com/gravitational/trace"
	yaml "gopkg.in/yaml.v2"
)

var infraInst runtimeinfra.Infra

type runtimeStateType struct {
	InfraName  runtimeinfra.InfraName  `yaml:"infra"`
	InfraState runtimeinfra.InfraState `yaml:"infra_state"`
}

func InitializeInfra() (err error) {
	state, err := loadRuntimeState()
	if err != nil {
		return trace.Wrap(err)
	}

	if *configs.InfraToInit == "" && state == nil {
		return trace.BadParameter("infra must be initialized first")
	}

	if *configs.InfraToInit != "" && state != nil {
		return trace.BadParameter("cannot initialize new infra on top of existing, please clean up the state and try again")
	}

	if state != nil {
		switch state.InfraName {
		case runtimeinfra.InfraNameLocal:
			infraInst, err = runtimeinfra.NewLocalFromState(state.InfraState)
		case runtimeinfra.InfraNameRemote:
			infraInst, err = runtimeinfra.NewRemoteFromState(state.InfraState)
		default:
			return trace.BadParameter("unknown infra type %s", state.InfraName)
		}

		if err != nil {
			return trace.Wrap(err)
		}

		return nil
	}

	switch runtimeinfra.InfraName(*configs.InfraToInit) {
	case runtimeinfra.InfraNameLocal:
		infraInst, err = runtimeinfra.NewLocal()
	case runtimeinfra.InfraNameRemote:
		infraInst, err = runtimeinfra.NewRemote()
	default:
		return trace.BadParameter("unknown infra type %s", *configs.InfraToInit)
	}

	if err != nil {
		return trace.Wrap(err)
	}

	return infraInst.Init()
}

func SaveState() error {
	if infraInst == nil {
		log.Infof("cluster inactive: skip UpdateState")
		return nil
	}

	var infraState = runtimeinfra.GetState(infraInst)

	var runtimeState = runtimeStateType{
		InfraName:  infraInst.GetName(),
		InfraState: infraState,
	}

	file, err := os.Create(*configs.StateConfigFile)
	if err != nil {
		return trace.Wrap(err)
	}
	defer file.Close()

	out, err := yaml.Marshal(runtimeState)
	if err != nil {
		return trace.Wrap(err)
	}

	_, err = file.Write(out)
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}

func Destroy() (err error) {
	if infraInst != nil {
		infraInst.Close()
		if *configs.Teardown == true {
			err = infraInst.Destroy()
			if err != nil {
				return trace.Wrap(err, "failed to destory an infra")
			}
		}
	}

	if *configs.Teardown == true {
		err = os.Remove(*configs.StateConfigFile)
		if err != nil && !os.IsNotExist(err) {
			return trace.Wrap(err, "failed to remove state file %q: %v", *configs.StateConfigFile)
		}
	}

	return nil
}

func loadRuntimeState() (*runtimeStateType, error) {
	confFile, err := os.Open(*configs.StateConfigFile)
	withExplanation := func(err error) error {
		return trace.Wrap(err, "failed to read infra state from %v file: ", *configs.StateConfigFile)
	}

	if err != nil && !os.IsNotExist(err) {
		return nil, withExplanation(err)
	}

	if err != nil {
		// No test state configuration
		return nil, nil
	}

	defer confFile.Close()

	var runtimeState = runtimeStateType{}

	configBytes, err := ioutil.ReadAll(confFile)
	if err != nil {
		return nil, withExplanation(err)
	}

	yaml.Unmarshal(configBytes, &runtimeState)
	if err != nil {
		return nil, withExplanation(err)
	}

	return &runtimeState, nil
}

func failedToLoad(err error) {
	Failf("failed to read infra state from %v file: ", *configs.StateConfigFile, trace.UserMessage(err))
}

func Failf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Error(msg)
	panic("Initialization error")
}
