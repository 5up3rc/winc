package main_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	testhelpers "code.cloudfoundry.org/winc/integration/helpers"
	"code.cloudfoundry.org/winc/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const (
	defaultTimeout  = time.Second * 10
	defaultInterval = time.Millisecond * 200
)

var (
	wincBin           string
	wincNetworkBin    string
	grootBin          string
	grootImageStore   string
	serverBin         string
	netoutBin         string
	clientBin         string
	rootfsURI         string
	helpers           *testhelpers.Helpers
	snetworkTempDir   string
	networkConfigFile string
	snetworkConfig    network.Config
)

func TestWincNetwork(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(defaultTimeout)
	SetDefaultEventuallyPollingInterval(defaultInterval)
	RunSpecs(t, "Winc-Network Suite")
}

var _ = BeforeSuite(func() {
	rand.Seed(time.Now().UnixNano() + int64(GinkgoParallelNode()))

	var present bool
	rootfsURI, present = os.LookupEnv("WINC_TEST_ROOTFS")
	Expect(present).To(BeTrue(), "WINC_TEST_ROOTFS not set")

	grootBin, present = os.LookupEnv("GROOT_BINARY")
	Expect(present).To(BeTrue(), "GROOT_BINARY not set")

	grootImageStore, present = os.LookupEnv("GROOT_IMAGE_STORE")
	Expect(present).To(BeTrue(), "GROOT_IMAGE_STORE not set")

	var err error
	wincBin, err = gexec.Build("code.cloudfoundry.org/winc/cmd/winc")
	Expect(err).ToNot(HaveOccurred())

	wincNetworkBin, err = gexec.Build("code.cloudfoundry.org/winc/cmd/winc-network")
	Expect(err).ToNot(HaveOccurred())

	wincNetworkDir := filepath.Dir(wincNetworkBin)
	o, err := exec.Command("gcc.exe", "-c", "..\\..\\network\\firewall\\dll\\firewall.c", "-o", filepath.Join(wincNetworkDir, "firewall.o")).CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), string(o))

	err = exec.Command("gcc.exe",
		"-shared",
		"-o", filepath.Join(wincNetworkDir, "firewall.dll"),
		filepath.Join(wincNetworkDir, "firewall.o"),
		"-lole32", "-loleaut32").Run()
	Expect(err).NotTo(HaveOccurred())

	serverBin, err = gexec.Build("code.cloudfoundry.org/winc/integration/winc-network/fixtures/server")
	Expect(err).ToNot(HaveOccurred())

	netoutBin, err = gexec.Build("code.cloudfoundry.org/winc/integration/winc-network/fixtures/netout")
	Expect(err).ToNot(HaveOccurred())

	clientBin, err = gexec.Build("code.cloudfoundry.org/winc/integration/winc-network/fixtures/client")
	Expect(err).ToNot(HaveOccurred())

	helpers = testhelpers.NewHelpers(wincBin, grootBin, grootImageStore, wincNetworkBin)

	tempDir, err = ioutil.TempDir("", "winc-network.config")
	Expect(err).NotTo(HaveOccurred())
	networkConfigFile = filepath.Join(tempDir, "winc-network.json")

	snetworkConfig = helpers.GenerateNetworkConfig()
	fmt.Printf("%+v\n", snetworkConfig)
	helpers.CreateNetwork(snetworkConfig, networkConfigFile)
})

var _ = AfterSuite(func() {
	helpers.DeleteNetwork(snetworkConfig, networkConfigFile)
	gexec.CleanupBuildArtifacts()
})
