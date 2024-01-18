/*
Copyright 2019 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cruntime

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"

	"k8s.io/minikube/pkg/minikube/bootstrapper/images"
	"k8s.io/minikube/pkg/minikube/command"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/style"
	"k8s.io/minikube/pkg/minikube/sysinit"
)

// Porto contains porto runtime state
type Porto struct {
	Socket            string
	Runner            CommandRunner
	ImageRepository   string
	KubernetesVersion semver.Version
	Init              sysinit.Manager
	InsecureRegistry  []string
}

// Name is a human readable name for porto
func (r *Porto) Name() string {
	return "porto"
}

// Style is the console style for porto
func (r *Porto) Style() style.Enum {
	return style.Porto
}

// parsePortoVersion parses version from portod --version
func parsePortoVersion(line string) (string, error) {
	// version: 5.3.30-alpha.7  /usr/sbin/portod
	// running: 5.3.30-alpha.7  /usr/sbin/portod
	rg := regexp.MustCompile(`(\d\.\S*)`)
	for _, v := range rg.FindStringSubmatch(line) {
		return v, nil
	}
	return "", fmt.Errorf("unknown version: %q", line)
}

// Version retrieves the current version of this runtime
func (r *Porto) Version() (string, error) {
	c := exec.Command("portod", "version")
	rr, err := r.Runner.RunCmd(c)
	if err != nil {
		return "", errors.Wrapf(err, "porto check version")
	}
	version, err := parsePortoVersion(rr.Stdout.String())
	if err != nil {
		return "", err
	}
	return version, nil
}

// SocketPath returns the path to the socket file for porto
func (r *Porto) SocketPath() string {
	if r.Socket != "" {
		return r.Socket
	}
	return "/run/portoshim.sock"
}

// Active returns if porto is active on the host
func (r *Porto) Active() bool {
	return r.Init.Active("porto")
}

// Available returns an error if it is not possible to use this runtime on a host
func (r *Porto) Available() error {
	c := exec.Command("which", "portoshim")
	if _, err := r.Runner.RunCmd(c); err != nil {
		return errors.Wrap(err, "check porto availability")
	}
	return checkCNIPlugins(r.KubernetesVersion)
}

// generatePortoConfig sets up /etc/porto/config.toml & /etc/porto/porto.conf.d/02-porto.conf
func generatePortoConfig(cr CommandRunner, imageRepository string, kv semver.Version, cgroupDriver string, insecureRegistry []string, inUserNamespace bool) error {
	return nil
}

// Enable idempotently enables porto on a host
func (r *Porto) Enable(disOthers bool, cgroupDriver string, inUserNamespace bool) error {
	if inUserNamespace {
		if err := CheckKernelCompatibility(r.Runner, 5, 11); err != nil {
			// For using overlayfs
			return fmt.Errorf("kernel >= 5.11 is required for rootless mode: %w", err)
		}
		if err := CheckKernelCompatibility(r.Runner, 5, 13); err != nil {
			// For avoiding SELinux error with overlayfs
			klog.Warningf("kernel >= 5.13 is recommended for rootless mode %v", err)
		}
	}
	if disOthers {
		if err := disableOthers(r, r.Runner); err != nil {
			klog.Warningf("disableOthers: %v", err)
		}
	}
	if err := populateCRIConfig(r.Runner, r.SocketPath()); err != nil {
		return err
	}

	if err := generatePortoConfig(r.Runner, r.ImageRepository, r.KubernetesVersion, cgroupDriver, r.InsecureRegistry, inUserNamespace); err != nil {
		return err
	}
	if err := enableIPForwarding(r.Runner); err != nil {
		return err
	}
	if err := r.Init.Restart("porto"); err != nil {
		return err
	}

	// HACK(ernado): porto is missing this image for some reason.
	if err := r.PullImage("registry.k8s.io/pause:3.7"); err != nil {
		return errors.Wrap(err, "pulling pause image")
	}

	return nil
}

// Disable idempotently disables porto on a host
func (r *Porto) Disable() error {
	return r.Init.ForceStop("porto")
}

// ImageExists checks if image exists based on image name and optionally image sha
func (r *Porto) ImageExists(name string, sha string) bool {
	klog.Infof("Checking existence of image with name %q and sha %q", name, sha)
	c := exec.Command("sudo", "portoctl", "docker-images")
	// note: image name and image id's sha can be on different lines
	// TODO(ernado): RLY?
	if rr, err := r.Runner.RunCmd(c); err != nil ||
		!strings.Contains(rr.Output(), name) ||
		(sha != "" && !strings.Contains(rr.Output(), sha)) {
		return false
	}
	return true
}

// ListImages lists images managed by this container runtime
func (r *Porto) ListImages(ListImagesOptions) ([]ListImage, error) {
	return listCRIImages(r.Runner)
}

// LoadImage loads an image into this runtime
func (r *Porto) LoadImage(path string) error {
	return errors.New("not implemented")
}

// PullImage pulls an image into this runtime
func (r *Porto) PullImage(name string) error {
	return pullCRIImage(r.Runner, name)
}

// SaveImage save an image from this runtime
func (r *Porto) SaveImage(name string, path string) error {
	return errors.New("not implemented")
}

// RemoveImage removes a image
func (r *Porto) RemoveImage(name string) error {
	return removeCRIImage(r.Runner, name)
}

// TagImage tags an image in this runtime
func (r *Porto) TagImage(source string, target string) error {
	return errors.New("not implemented")
}

// BuildImage builds an image into this runtime
func (r *Porto) BuildImage(src string, file string, tag string, push bool, env []string, opts []string) error {
	return errors.New("not implemented")
}

// PushImage pushes an image
func (r *Porto) PushImage(name string) error {
	return errors.New("not implemented")
}

// CGroupDriver returns cgroup driver ("cgroupfs" or "systemd")
func (r *Porto) CGroupDriver() (string, error) {
	return "systemd", nil
}

// KubeletOptions returns kubelet options for a porto
func (r *Porto) KubeletOptions() map[string]string {
	return kubeletCRIOptions(r, r.KubernetesVersion)
}

// ListContainers returns a list of managed by this container runtime
func (r *Porto) ListContainers(o ListContainersOptions) ([]string, error) {
	return listCRIContainers(r.Runner, "", o)
}

// PauseContainers pauses a running container based on ID
func (r *Porto) PauseContainers(ids []string) error {
	return pauseCRIContainers(r.Runner, "", ids)
}

// UnpauseContainers unpauses a running container based on ID
func (r *Porto) UnpauseContainers(ids []string) error {
	return unpauseCRIContainers(r.Runner, "", ids)
}

// KillContainers removes containers based on ID
func (r *Porto) KillContainers(ids []string) error {
	return killCRIContainers(r.Runner, ids)
}

// StopContainers stops containers based on ID
func (r *Porto) StopContainers(ids []string) error {
	return stopCRIContainers(r.Runner, ids)
}

// ContainerLogCmd returns the command to retrieve the log for a container based on ID
func (r *Porto) ContainerLogCmd(id string, len int, follow bool) string {
	return criContainerLogCmd(r.Runner, id, len, follow)
}

// SystemLogCmd returns the command to retrieve system logs
func (r *Porto) SystemLogCmd(len int) string {
	return fmt.Sprintf("sudo tail -n %d /var/log/portod.log", len)
}

// Preload preloads the container runtime with k8s images
func (r *Porto) Preload(cc config.ClusterConfig) error {
	k8sVersion := cc.KubernetesConfig.KubernetesVersion
	imageList, err := images.Kubeadm(cc.KubernetesConfig.ImageRepository, k8sVersion)
	if err != nil {
		return errors.Wrap(err, "getting images")
	}
	if portoImagesPreloaded(r.Runner, imageList) {
		klog.Info("Images already preloaded, skipping extraction")
		return nil
	}
	for _, img := range imageList {
		if err := r.PullImage(img); err != nil {
			return errors.Wrapf(err, "pulling image %q", img)
		}
	}
	return r.Restart()
}

// Restart restarts this container runtime on a host
func (r *Porto) Restart() error {
	return r.Init.Restart("porto")
}

// portoImagesPreloaded returns true if all images have been preloaded
func portoImagesPreloaded(runner command.Runner, images []string) bool {
	rr, err := runner.RunCmd(exec.Command("sudo", "crictl", "images", "--output", "json"))
	if err != nil {
		return false
	}

	var jsonImages crictlImages
	err = json.Unmarshal(rr.Stdout.Bytes(), &jsonImages)
	if err != nil {
		klog.Errorf("failed to unmarshal images, will assume images are not preloaded")
		return false
	}

	// Make sure images == imgs
	for _, i := range images {
		found := false
		for _, ji := range jsonImages.Images {
			for _, rt := range ji.RepoTags {
				i = addRepoTagToImageName(i)
				if i == rt {
					found = true
					break
				}
			}
			if found {
				break
			}

		}
		if !found {
			klog.Infof("couldn't find preloaded image for %q. assuming images are not preloaded.", i)
			return false
		}
	}
	klog.Infof("all images are preloaded for porto runtime.")
	return true
}

// ImagesPreloaded returns true if all images have been preloaded
func (r *Porto) ImagesPreloaded(images []string) bool {
	return portoImagesPreloaded(r.Runner, images)
}
