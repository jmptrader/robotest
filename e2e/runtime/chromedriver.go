package runtime

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/robotest/e2e/runtime/configs"
	"github.com/gravitational/trace"

	web "github.com/sclevine/agouti"
)

// driver is a test-global web driver instance
var driver *web.WebDriver

// CreateDriver creates a new instance of the web driver
func CreateDriver() error {
	runtimeCfg := configs.GetRuntime()
	if runtimeCfg.WebDriverURL != "" {
		log.Debugf("WebDriverURL specified - skip CreateDriver")
		return nil
	}
	driver = web.ChromeDriver()

	if driver == nil {
		return trace.Errorf("cannot create chromedriver")
	}

	return driver.Start()
}

// CloseDriver stops and closes the test-global web driver
func CloseDriver() {
	if driver != nil {
		driver.Stop()
	}
}

func newPage() (*web.Page, error) {
	var cfg = configs.GetRuntime()
	if cfg.WebDriverURL != "" {
		return web.NewPage(cfg.WebDriverURL, web.Desired(web.Capabilities{
			"chromeOptions": map[string][]string{
				"args": []string{
					// There is no GPU inside docker box!
					"disable-gpu",
					// Sandbox requires namespace permissions that we don't have on a container
					"no-sandbox",
				},
			},
			"javascriptEnabled": true,
		}))
	}
	return driver.NewPage()
}
