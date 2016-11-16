package infra

import (
	"fmt"
	"io"
	"net/url"
	"os"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/trace"
)

func New(conf Config) (Infra, error) {
	return &staticCluster{
		nodes:        conf.InitialCluster,
		opsCenterURL: conf.OpsCenterURL,
	}, nil
}

func NewWizard(conf Config, provisioner Provisioner) (Infra, *ProvisionerOutput, error) {
	output, err := startWizard(provisioner)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	return &wizardCluster{
		provisioner:       provisioner,
		ProvisionerOutput: *output,
	}, output, nil
}

type Provisioner interface {
	Create() (*ProvisionerOutput, error)
	Destroy() error
	Connect(addr string) (*ssh.Session, error)
	// SelectInterface returns the index (in addrs) of network address to use for
	// installation.
	// addrs is guaranteed to have at least one element
	SelectInterface(output ProvisionerOutput, addrs []string) (int, error)
	StartInstall(session *ssh.Session) error
	Nodes() []Node
	// Allocate allocates a new node (from the pool of available nodes)
	// and returns a reference to it
	Allocate() (Node, error)
	// Deallocate places specified node back to the node pool
	Deallocate(Node) error
}

type Infra interface {
	Nodes() []Node
	NumNodes() int
	OpsCenterURL() string
	// Close closes the cluster resources
	Close() error
	// Run runs the specified command on all active nodes in the cluster
	Run(command string) error
	// Allocate(addr, user, key string) error
	// Deallocate(addr string) error
}

type Node interface {
	Run(command string, output io.Writer) error
	Connect() (*ssh.Session, error)
}

type ProvisionerOutput struct {
	InstallerIP  string
	PrivateIPs   []string
	PublicIPs    []string
	InstallerURL url.URL
}

func (r ProvisionerOutput) String() string {
	return fmt.Sprintf("ProvisionerOutput(installer IP=%v, private IPs=%v, public IPs=%v)",
		r.InstallerIP, r.PrivateIPs, r.PublicIPs)
}

func RunOnNodes(command string, nodes []Node) error {
	errCh := make(chan error, len(nodes))
	for _, node := range nodes {
		go func(errCh chan<- error) {
			errCh <- node.Run(command, os.Stderr)
		}(errCh)
	}
	var errors []error
	for err := range errCh {
		if err != nil {
			errors = append(errors, err)
		}
	}
	return trace.NewAggregate(errors...)
}
