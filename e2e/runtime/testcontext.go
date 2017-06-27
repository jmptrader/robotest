package runtime

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gravitational/robotest/e2e/runtime/configs"
	runtimeinfra "github.com/gravitational/robotest/e2e/runtime/infra"
	"github.com/gravitational/robotest/infra"

	"github.com/gravitational/trace"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
)

// T defines a framework type.
// Framework stores attributes common to a single context
type TContext struct {
	Page *web.Page
}

type backupFlag bool

const (
	withBackup    backupFlag = true
	withoutBackup backupFlag = false
)

// New creates a new instance of the framework.
// Creating a framework instance installs a set of BeforeEach/AfterEach to
// emulate BeforeAll/AfterAll for controlled access to resources that should
// only be created once per context
func New() *TContext {
	tcontext := &TContext{}
	BeforeEach(tcontext.BeforeEach)
	return tcontext
}

// BeforeEach emulates BeforeAll for a context.
// It creates a new web page that is only initialized once per series of It
// grouped in any given context
func (r *TContext) BeforeEach() {
	if r.Page == nil {
		var err error
		r.Page, err = newPage()
		Expect(err).NotTo(HaveOccurred())
	}
}

// Distribute executes the specified command on nodes
func (r *TContext) Distribute(command string, nodes ...infra.Node) {
	ensureInfra()
	Expect(infraInst.Provisioner()).NotTo(BeNil(), "requires a provisioner")
	if len(nodes) == 0 {
		nodes = infraInst.Provisioner().NodePool().AllocatedNodes()
	}
	Expect(infra.Distribute(command, nodes...)).To(Succeed())
}

// RunAgentCommand interprets the specified command as agent command.
// It will modify the agent command line to start agent in background
// and will distribute the command on the specified nodes
func (r *TContext) RunAgentCommand(command string, nodes ...infra.Node) {
	command, err := infra.ConfigureAgentCommandRunDetached(command)
	Expect(err).NotTo(HaveOccurred())
	r.Distribute(command, nodes...)
}

func (r *TContext) GetInfra() runtimeinfra.Infra {
	ensureInfra()
	return infraInst
}

func (r *TContext) BindCfg(object interface{}, path string) {
	// ignore parsing if test is to be skipped
	if !r.needsEvaluation(path) {
		return
	}

	configs.Parse(object, path)
	err := configs.Validate(object)
	if err != nil {
		Failf("failed to validate %q section from config file: %v", path, trace.UserMessage(err))
	}
}

func (r *TContext) SetNewOpsCenterURL(entryURL string) {
	ensureInfra()
	newURL, err := r.appendToBase(entryURL, "web/portal")
	Expect(err).NotTo(HaveOccurred())
	infraInst.SetOpsCenterURL(newURL)
}

func (r *TContext) GetWizardURL() string {
	ensureInfra()
	return infraInst.GetWizardURL()
}

func (r *TContext) GetLogin() configs.Login {
	runtimeCfg := configs.GetRuntime()
	return runtimeCfg.Login
}

func (r *TContext) GetClusterURL(clusterName string) string {
	ensureInfra()
	suffix := fmt.Sprintf("web/site/%v", clusterName)
	urlString, err := r.appendToBase(infraInst.GetOpsCenterURL(), suffix)
	Expect(err).NotTo(HaveOccurred(), "should resolve cluster URL from infra")
	return urlString
}

func (r *TContext) needsEvaluation(testPath string) bool {
	return strings.Contains(*configs.FlagTests2Run, testPath)
}

func (r *TContext) appendToBase(base string, path string) (string, error) {
	baseUrl, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	baseUrl.RawQuery = ""
	baseUrl.Path = path
	return baseUrl.String(), nil
}

func ensureInfra() {
	Expect(infraInst).NotTo(BeNil(), "requires an infra")
}
