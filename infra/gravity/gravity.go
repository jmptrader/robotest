package gravity

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/gravitational/robotest/infra"
	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/trace"

	"golang.org/x/crypto/ssh"
)

// Gravity is interface to remote gravity CLI
type Gravity interface {
	// Install operates on initial master node
	Install(ctx context.Context, param InstallCmd) error
	// Retrieve status
	Status(ctx context.Context) (*GravityStatus, error)
	// OfflineUpdate tries to upgrade application version
	OfflineUpdate(ctx context.Context, installerUrl string) error
	// Join asks to join existing cluster (or installation in progress)
	Join(ctx context.Context, param JoinCmd) error
	// Node returns underlying VM instance
	Node() infra.Node
	// Client returns SSH client to VM instance
	Client() *ssh.Client
}

// InstallCmd install parameters passed to first node
type InstallCmd struct {
	// Token is required to join cluster
	Token string
	// Cluster is Optional name of the cluster. Autogenerated if not set.
	Cluster string
	// Flavor is (Optional) Application flavor. See Application Manifest for details.
	Flavor string
	// K8SConfig is (Optional) File with Kubernetes resources to create in the cluster during installation.
	K8SConfig string
	// PodNetworkCidr is (Optional) CIDR range Kubernetes will be allocating node subnets and pod IPs from. Must be a minimum of /16 so Kubernetes is able to allocate /24 to each node. Defaults to 10.244.0.0/16.
	PodNetworkCIDR string
	// ServiceCidr (Optional) CIDR range Kubernetes will be allocating service IPs from. Defaults to 10.100.0.0/16.
	ServiceCIDR string
}

// JoinCmd represents various parameters for Join
type JoinCmd struct {
	// InstallDir is set automatically
	InstallDir string
	// PeerAddr is other node (i.e. master)
	PeerAddr string
	Token    string
	Role     string
}

// GravityStatus is serialized form of `gravity status` CLI.
type GravityStatus struct {
	Application string
	Cluster     string
	Status      string
	// Token is secure token which prevents rogue nodes from joining the cluster during installation.
	Token string `validation:"required"`
	// Nodes defines nodes the cluster observes
	Nodes []string
}

type gravity struct {
	node         infra.Node
	logFn        sshutils.LogFnType
	installDir   string
	dockerDevice string
	ssh          *ssh.Client
}

const retrySSH = time.Second * 10

// FromNode takes a provisioned and set up Node and makes Gravity control interface
func fromNode(ctx context.Context, logFn sshutils.LogFnType, node infra.Node, installDir, dockerDevice string) (Gravity, error) {
	g := gravity{
		node:         node,
		logFn:        logFn,
		installDir:   installDir,
		dockerDevice: dockerDevice,
	}

	// node might be provisioned, but SSH daemon not up just yet
	for {
		client, err := node.Client()

		if err == nil {
			g.ssh = client
			return &g, nil
		}

		logFn("error SSH %s, retry in %v", node.Addr(), retrySSH)
		select {
		case <-ctx.Done():
			return nil, trace.Wrap(err, "SSH timed out dialing %s", node.Addr())
		case <-time.After(retrySSH):
		}
	}
}

func (g *gravity) Node() infra.Node {
	return g.node
}

func (g *gravity) Client() *ssh.Client {
	return g.ssh
}

// Install runs gravity install with params
func (g *gravity) Install(ctx context.Context, param InstallCmd) error {
	cmd := fmt.Sprintf("cd %s && sudo ./gravity install --advertise-addr=%s --token=%s --flavor=%s --docker-device=%s",
		g.installDir, g.node.PrivateAddr(), param.Token, param.Flavor, g.dockerDevice)

	err := sshutils.Run(ctx, g.logFn, g.ssh,
		cmd, nil)
	return trace.Wrap(err, cmd)
}

func (g *gravity) Status(ctx context.Context) (*GravityStatus, error) {
	cmd := fmt.Sprintf("cd %s && sudo ./gravity status")
	status, exit, err := sshutils.RunAndParse(ctx, g.logFn, g.ssh,
		cmd, nil, parseStatus)

	if err != nil {
		return nil, trace.Wrap(err, cmd)
	}

	if exit != 0 {
		return nil, trace.Errorf("%s returned %d", cmd, exit)
	}

	return status.(*GravityStatus), nil
}

func (g *gravity) OfflineUpdate(ctx context.Context, installerUrl string) error {
	return nil
}

var joinCmdTemplate = template.Must(
	template.New("gravity_join").Parse(
		`cd {{.P.InstallDir}} && sudo ./gravity join {{.Cmd.PeerAddr}} \
		--advertise-addr={{.P.PrivateAddr}} --token={{.Cmd.Token}} \
		--role={{.Cmd.Role}} --docker-device={{.P.DockerDevice}}`))

type autoVals struct{ InstallDir, PrivateAddr, DockerDevice string }
type cmdEx struct {
	P   autoVals
	Cmd interface{}
}

func (g *gravity) Join(ctx context.Context, cmd JoinCmd) error {
	var buf bytes.Buffer
	err := joinCmdTemplate.Execute(&buf, cmdEx{
		P:   autoVals{g.installDir, g.Node().PrivateAddr(), g.dockerDevice},
		Cmd: cmd,
	})
	if err != nil {
		return trace.Wrap(err, buf.String())
	}

	err = sshutils.Run(ctx, g.logFn, g.ssh, buf.String(), nil)
	return trace.Wrap(err, cmd)
}