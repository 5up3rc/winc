package main_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	acl "github.com/hectane/go-acl"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/windows"
)

var _ = Describe("Events", func() {
	Context("given an existing container id", func() {
		var (
			containerId string
			bundlePath  string
			bundleSpec  specs.Spec
		)

		BeforeEach(func() {
			var err error
			bundlePath, err = ioutil.TempDir("", "winccontainer")
			Expect(err).To(Succeed())

			containerId = filepath.Base(bundlePath)

			bundleSpec = helpers.GenerateRuntimeSpec(helpers.CreateVolume(rootfsURI, containerId))
			bundleSpec.Mounts = []specs.Mount{{Source: filepath.Dir(sleepBin), Destination: "C:\\tmp"}}
			Expect(acl.Apply(filepath.Dir(sleepBin), false, false, acl.GrantName(windows.GENERIC_ALL, "Everyone"))).To(Succeed())
			helpers.CreateContainer(bundleSpec, bundlePath, containerId)
		})

		AfterEach(func() {
			helpers.DeleteContainer(containerId)
			helpers.DeleteVolume(containerId)
			Expect(os.RemoveAll(bundlePath)).To(Succeed())
		})

		Context("when the container has been created", func() {
			It("exits without error", func() {
				cmd := exec.Command(wincBin, "events", containerId)
				stdOut, stdErr, err := helpers.Execute(cmd)
				Expect(err).NotTo(HaveOccurred(), stdOut.String(), stdErr.String())
			})

			Context("when passed the --stats flag", func() {
				BeforeEach(func() {
					pid := helpers.GetContainerState(containerId).Pid
					helpers.CopyFile(filepath.Join("c:\\", "proc", strconv.Itoa(pid), "root", "consume.exe"), consumeBin)
				})

				It("prints the container memory stats to stdout", func() {
					stats := getStats(containerId)
					Expect(stats.Data.Memory.Stats.TotalRss).To(BeNumerically(">", 0))

					memConsumedBytes := 200 * 1024 * 1024

					cmd := exec.Command(wincBin, "exec", containerId, "c:\\consume.exe", strconv.Itoa(memConsumedBytes), "10")
					stdOut, err := cmd.StdoutPipe()
					Expect(err).NotTo(HaveOccurred())

					Expect(cmd.Start()).To(Succeed())

					Eventually(func() string {
						out := make([]byte, 256, 256)
						n, _ := stdOut.Read(out)
						return strings.TrimSpace(string(out[:n]))
					}).Should(Equal(fmt.Sprintf("Allocated %d", memConsumedBytes)))

					statsAfter := getStats(containerId)
					goRuntimeOverhead := uint64(25 * 1024 * 1024)
					expectedMemConsumedBytes := stats.Data.Memory.Stats.TotalRss + uint64(memConsumedBytes) + goRuntimeOverhead
					threshold := 30 * 1024 * 1024
					Expect(statsAfter.Data.Memory.Stats.TotalRss).To(BeNumerically("~", expectedMemConsumedBytes, threshold))
					Expect(cmd.Wait()).To(Succeed())
				})

				It("prints the container CPU stats to stdout", func() {
					cpuUsageBefore := getStats(containerId).Data.CPUStats.CPUUsage.Usage
					Expect(cpuUsageBefore).To(BeNumerically(">", 0))

					args := []string{"powershell.exe", "-Command", "$result = 1; foreach ($number in 1..2147483647) {$result = $result * $number};"}
					stdOut, stdErr, err := helpers.ExecInContainer(containerId, args, true)
					Expect(err).ToNot(HaveOccurred(), stdOut.String(), stdErr.String())

					cpuUsageAfter := getStats(containerId).Data.CPUStats.CPUUsage.Usage
					Expect(cpuUsageAfter).To(BeNumerically(">", cpuUsageBefore))
				})
			})
		})
	})

	Context("given a nonexistent container id", func() {
		It("errors", func() {
			cmd := exec.Command(wincBin, "events", "doesntexist")
			stdOut, stdErr, err := helpers.Execute(cmd)
			Expect(err).To(HaveOccurred(), stdOut.String(), stdErr.String())

			Expect(stdErr.String()).To(ContainSubstring("container doesntexist encountered an error during OpenContainer"))
			Expect(stdErr.String()).To(ContainSubstring("A Compute System with the specified identifier does not exist"))
		})
	})
})
