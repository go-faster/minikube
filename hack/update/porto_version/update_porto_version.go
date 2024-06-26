/*
Copyright 2023 The Kubernetes Authors All rights reserved.

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

package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"k8s.io/minikube/hack/update"
)

var schema = map[string]update.Item{
	"deploy/iso/minikube-iso/arch/x86_64/package/porto-bin/porto-bin.mk": {
		Replace: map[string]string{
			`PORTO_BIN_VERSION = .*`: `PORTO_BIN_VERSION = {{.Version}}`,
			`PORTO_BIN_COMMIT = .*`:  `PORTO_BIN_COMMIT = {{.Commit}}`,
		},
	},
}

type Data struct {
	Version string
	Commit  string
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Get edge version, since most of go-faster releases are `.alpha-vX`, recognized as edge.
	_, _, edge, err := update.GHReleases(ctx, "go-faster", "porto")
	if err != nil {
		klog.Fatalf("Unable to get edge version: %v", err)
	}

	version := edge.Tag
	data := Data{Version: version, Commit: edge.Commit}
	update.Apply(schema, data)

	if err := updateHashFile(version, "amd64", "x86_64/package/porto-bin"); err != nil {
		klog.Fatalf("failed updating amd64 hash file: %v", err)
	}
}

func updateHashFile(version, arch, packagePath string) error {
	// https://github.com/go-faster/porto/releases/download/v5.3.31/porto_focal_v5.3.31_amd64.tgz
	link := fmt.Sprintf("https://github.com/go-faster/porto/releases/download/%[1]s/porto_focal_%[1]s_%[2]s.tgz", version, arch)
	r, err := http.Get(link)
	if err != nil {
		return fmt.Errorf("failed to download binary: %v", err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download binary: %s", r.Status)
	}
	h := sha256.New()
	if _, err := io.Copy(h, r.Body); err != nil {
		return fmt.Errorf("failed to copy response body: %v", err)
	}
	filePath := fmt.Sprintf("../../../deploy/iso/minikube-iso/arch/%s/porto-bin.hash", packagePath)
	b, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read hash file: %v", err)
	}
	if strings.Contains(string(b), version) {
		klog.Infof("hash file already contains %q", version)
		return nil
	}
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open hash file: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(fmt.Sprintf("sha256 %x  porto_focal_%s_%s.tgz\n", h.Sum(nil), version, arch)); err != nil {
		return fmt.Errorf("failed to write to hash file: %v", err)
	}
	return nil
}
