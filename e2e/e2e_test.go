package e2e

import (
	"testing"
	"time"

	"github.com/gravitational/robotest/e2e/runtime"
	"github.com/gravitational/robotest/e2e/runtime/configs"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/gomega"
)

var delayedSpecs = []func(){}

type duration time.Duration

// Duration returns this duration as time.Duration
func (r duration) Duration() time.Duration {
	return time.Duration(r)
}

func RoboDescribe(text string, body func()) bool {
	f := func() {
		ginkgo.Describe("[robotest] "+text, body)
	}

	delayedSpecs = append(delayedSpecs, f)
	return true
}

func TestE2E(t *testing.T) {
	config.GinkgoConfig.FocusString = *configs.FlagTests2Run
	if *configs.InfraToInit != "" {
		config.GinkgoConfig.SkipString = ".*"
	}

	gomega.RegisterFailHandler(ginkgo.Fail)

	for _, s := range delayedSpecs {
		s()
	}
	ginkgo.RunSpecs(t, "e2e suite")
}

// Run the tasks that are meant to be run once per invocation
var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {
	// Run only on ginkgo node 1
	gomega.Expect(runtime.CreateDriver()).NotTo(gomega.HaveOccurred())
	gomega.Expect(runtime.InitializeInfra()).NotTo(gomega.HaveOccurred())

	return nil
}, func([]byte) {
})

var _ = ginkgo.SynchronizedAfterSuite(func() {
	// Run on all ginkgo nodes
}, func() {
	runtime.CloseDriver()
	runtime.SaveState()
	gomega.Expect(runtime.Destroy()).NotTo(gomega.HaveOccurred())
})
