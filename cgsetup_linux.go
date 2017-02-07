package cgsetup

// Original source: https://github.com/cloudfoundry/guardian/blob/master/rundmc/starter.go
// code has been modified such to only depend on the Go standard library

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
)

const cgroupsHeader = "#subsys_name hierarchy num_cgroups enabled"

type CgroupSetupper struct {
	CgroupPath    string
	CommandRunner RealCommandRunner

	ProcCgroups     io.ReadCloser
	ProcSelfCgroups io.ReadCloser
}

func New(procCgroupReader io.ReadCloser, procSelfCgroupReader io.ReadCloser, cgroupMountpoint string, runner RealCommandRunner) *CgroupSetupper {
	return &CgroupSetupper{
		CgroupPath:      cgroupMountpoint,
		ProcCgroups:     procCgroupReader,
		ProcSelfCgroups: procSelfCgroupReader,
		CommandRunner:   runner,
	}
}

func (c *CgroupSetupper) EnsureCgroupsMounted() error {
	defer func() {
		c.ProcCgroups.Close()
		c.ProcSelfCgroups.Close()
	}()

	if err := os.MkdirAll(c.CgroupPath, 0755); err != nil {
		return err
	}

	if !c.isMountPoint(c.CgroupPath) {
		c.mountTmpfsOnCgroupPath(c.CgroupPath)
	} else {
		log(fmt.Sprintf("cgroups tmpfs already mounted at %s", c.CgroupPath))
	}

	subsystemGroupings, err := c.subsystemGroupings()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(c.ProcCgroups)

	if !scanner.Scan() {
		return CgroupsFormatError{Content: "(empty)"}
	}

	if _, err := fmt.Sscanf(scanner.Text(), cgroupsHeader); err != nil {
		return CgroupsFormatError{Content: scanner.Text()}
	}

	for scanner.Scan() {
		var subsystem string
		var skip, enabled int
		n, err := fmt.Sscanf(scanner.Text(), "%s %d %d %d ", &subsystem, &skip, &skip, &enabled)
		if err != nil || n != 4 {
			return CgroupsFormatError{Content: scanner.Text()}
		}

		if enabled == 0 {
			continue
		}

		cgroupsToMount, found := subsystemGroupings[subsystem]
		if !found {
			cgroupsToMount = subsystem
		}

		if err := c.mountCgroup(path.Join(c.CgroupPath, subsystem), cgroupsToMount); err != nil {
			return err
		}
	}

	return nil
}

type CgroupsFormatError struct {
	Content string
}

func (err CgroupsFormatError) Error() string {
	return fmt.Sprintf("unknown /proc/cgroups format: %s", err.Content)
}

func log(msg string) {
	fmt.Printf("%s\n", msg)
}

func (c *CgroupSetupper) isMountPoint(path string) bool {
	// append trailing slash to force symlink traversal; symlinking e.g. 'cpu'
	// to 'cpu,cpuacct' is common
	return c.CommandRunner.Run(exec.Command("mountpoint", "-q", path+"/")) == nil
}

func (c *CgroupSetupper) mountTmpfsOnCgroupPath(path string) {
	log(fmt.Sprintf("mounting tmpfs on cgroup path at %s", path))

	if err := c.CommandRunner.Run(exec.Command("mount", "-t", "tmpfs", "-o", "uid=0,gid=0,mode=0755", "cgroup", path)); err != nil {
		log(fmt.Sprintf("ERROR: %s", err.Error()))
	} else {
		log(fmt.Sprintf("mounted tmpfs on cgroup path at %s", path))
	}
}

func (c *CgroupSetupper) subsystemGroupings() (map[string]string, error) {
	groupings := map[string]string{}

	scanner := bufio.NewScanner(c.ProcSelfCgroups)

	for scanner.Scan() {
		segs := strings.Split(scanner.Text(), ":")
		if len(segs) != 3 {
			continue
		}

		subsystems := strings.Split(segs[1], ",")
		for _, subsystem := range subsystems {
			groupings[subsystem] = segs[1]
		}
	}

	return groupings, scanner.Err()
}

func (c *CgroupSetupper) mountCgroup(cgroupPath, subsystems string) error {
	log(fmt.Sprintf("mounting cgroup %s at %s", subsystems, cgroupPath))

	if !c.isMountPoint(cgroupPath) {
		if err := os.MkdirAll(cgroupPath, 0755); err != nil {
			return fmt.Errorf("mkdir '%s': %s", cgroupPath, err)
		}

		cmd := exec.Command("mount", "-n", "-t", "cgroup", "-o", subsystems, "cgroup", cgroupPath)
		cmd.Stderr = os.Stderr
		if err := c.CommandRunner.Run(cmd); err != nil {
			return fmt.Errorf("mounting subsystems '%s' in '%s': %s", subsystems, cgroupPath, err)
		}
	} else {
		log(fmt.Sprintf("subsystem %s already mounted at %s", subsystems, cgroupPath))
	}

	log(fmt.Sprintf("mounted cgroup %s at %s", subsystems, cgroupPath))

	return nil
}
