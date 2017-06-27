package bandwagon

import (
	"strings"

	"github.com/gravitational/robotest/e2e/uimodel/defaults"
	"github.com/gravitational/robotest/e2e/uimodel/utils"

	log "github.com/Sirupsen/logrus"
	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

// Bandwagon is bandwagon ui model
type Bandwagon struct {
	page *web.Page
}

type BandwagonForm struct {
	Organization string `yaml:"organization"  `
	Password     string `yaml:"password" validate:"required"`
	Username     string `yaml:"username" validate:"required"`
	RemoteAccess bool   `yaml:"remote_access" `
}

// Open navigates to bandwagon URL and returns its ui model
func Open(page *web.Page, domainName string) Bandwagon {
	url := utils.GetSiteURL(page, domainName)
	Expect(page.Navigate(url)).To(Succeed(), "navigating to bandwagon")
	Eventually(page.FindByClass("my-page-btn-submit"), defaults.AppLoadTimeout).
		Should(BeFound(), "should wait for bandwagon to load")
	return Bandwagon{page}
}

// SubmitForm submits bandwagon form
func (b *Bandwagon) SubmitForm(form BandwagonForm) {
	log.Infof("trying to submit bandwagon form")
	log.Infof("entering email: %s", form.Username)
	Expect(b.page.FindByName("email").Fill(form.Username)).To(Succeed(), "should enter email")
	count, _ := b.page.FindByName("name").Count()
	if count > 0 {
		log.Infof("entering username: %s", form.Username)
		Expect(b.page.FindByName("name").Fill(form.Username)).To(Succeed(), "should enter username")
	}

	log.Infof("entering password: %s", form.Password)
	Expect(b.page.FindByName("password").Fill(form.Password)).To(Succeed(), "should enter password")
	Expect(b.page.FindByName("passwordConfirmed").Fill(form.Password)).
		To(Succeed(), "should re-enter password")

	log.Infof("specifying remote access")
	utils.SelectRadio(b.page, ".my-page-section .grv-control-radio", func(value string) bool {
		prefix := "Disable remote"
		if form.RemoteAccess {
			prefix = "Enable remote"
		}
		return strings.HasPrefix(value, prefix)
	})

	log.Infof("submitting the form")
	Expect(b.page.FindByClass("my-page-btn-submit").Click()).To(Succeed(), "should click submit button")

	utils.PauseForPageJs()
	Eventually(func() bool {
		return utils.IsFound(b.page, ".my-page-btn-submit .fa-spin")
	}, defaults.BandwagonSubmitFormTimeout).Should(BeFalse(), "wait for progress indicator to disappear")
}
