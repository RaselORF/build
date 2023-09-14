// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package task

import (
	"context"
	"fmt"
	"io/fs"
	"math/rand"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"cloud.google.com/go/storage"
	"golang.org/x/build/internal/gcsfs"
)

type CloudBuildClient interface {
	// RunBuildTrigger runs an existing trigger in project with the given
	// substitutions.
	RunBuildTrigger(ctx context.Context, project, trigger string, substitutions map[string]string) (CloudBuild, error)
	// RunScript runs the given script under bash -eux -o pipefail in
	// ScriptProject. Outputs are collected into the build's ResultURL,
	// readable with ResultFS. The script will have the latest version of Go
	// on $PATH.
	// If gerritProject is specified, the script will run in the root of a
	// checkout of the tip version of that repository.
	RunScript(ctx context.Context, script string, gerritProject string, outputs []string) (CloudBuild, error)
	// Completed reports whether a build has finished, returning an error if
	// it's failed. It's suitable for use with AwaitCondition.
	Completed(ctx context.Context, build CloudBuild) (detail string, completed bool, _ error)
	// ResultFS returns an FS that contains the results of the given build.
	ResultFS(ctx context.Context, build CloudBuild) (fs.FS, error)
}

type RealCloudBuildClient struct {
	BuildClient   *cloudbuild.Client
	StorageClient *storage.Client
	ScriptProject string
	ScratchURL    string
}

// CloudBuild represents a Cloud Build that can be queried with the status
// methods on CloudBuildClient.
type CloudBuild struct {
	Project, ID string
	ResultURL   string
}

func (c *RealCloudBuildClient) RunBuildTrigger(ctx context.Context, project, trigger string, substitutions map[string]string) (CloudBuild, error) {
	op, err := c.BuildClient.RunBuildTrigger(ctx, &cloudbuildpb.RunBuildTriggerRequest{
		ProjectId: project,
		TriggerId: trigger,
		Source: &cloudbuildpb.RepoSource{
			Substitutions: substitutions,
		},
	})
	if err != nil {
		return CloudBuild{}, err
	}
	if _, err = op.Poll(ctx); err != nil {
		return CloudBuild{}, err
	}
	meta, err := op.Metadata()
	if err != nil {
		return CloudBuild{}, err
	}
	return CloudBuild{Project: project, ID: meta.Build.Id}, nil
}

func (c *RealCloudBuildClient) RunScript(ctx context.Context, script string, gerritProject string, outputs []string) (CloudBuild, error) {
	const downloadGoScript = `#!/usr/bin/env bash
set -eux
archive=$(wget -qO - 'https://go.dev/dl/?mode=json' | grep -Eo 'go.*linux-amd64.tar.gz' | head -n 1)
wget -qO - https://go.dev/dl/${archive} | tar -xz
mv go released_go
`

	const scriptPrefix = `#!/usr/bin/env bash
set -eux
set -o pipefail
export PATH=$PWD/released_go/bin:$PATH
`

	resultURL := fmt.Sprintf("%v/script-build-%v", c.ScratchURL, rand.Int63())
	build := &cloudbuildpb.Build{
		Steps: []*cloudbuildpb.BuildStep{
			{
				Name:   "bash",
				Script: downloadGoScript,
			},
			{
				Name:   "bash",
				Script: scriptPrefix + script,
			},
		},
		Artifacts: &cloudbuildpb.Artifacts{
			Objects: &cloudbuildpb.Artifacts_ArtifactObjects{
				Location: resultURL,
				Paths:    outputs,
			},
		},
		Options: &cloudbuildpb.BuildOptions{
			MachineType: cloudbuildpb.BuildOptions_E2_HIGHCPU_8,
		},
	}
	if gerritProject != "" {
		build.Source = &cloudbuildpb.Source{
			Source: &cloudbuildpb.Source_GitSource{
				GitSource: &cloudbuildpb.GitSource{
					Url:      "https://go.googlesource.com/" + gerritProject,
					Revision: "HEAD",
				},
			},
		}
	}
	op, err := c.BuildClient.CreateBuild(ctx, &cloudbuildpb.CreateBuildRequest{
		ProjectId: c.ScriptProject,
		Build:     build,
	})
	if err != nil {
		return CloudBuild{}, err
	}
	if _, err = op.Poll(ctx); err != nil {
		return CloudBuild{}, err
	}
	meta, err := op.Metadata()
	if err != nil {
		return CloudBuild{}, err
	}
	return CloudBuild{Project: c.ScriptProject, ID: meta.Build.Id, ResultURL: resultURL}, nil

}

func (c *RealCloudBuildClient) Completed(ctx context.Context, build CloudBuild) (string, bool, error) {
	b, err := c.BuildClient.GetBuild(ctx, &cloudbuildpb.GetBuildRequest{
		ProjectId: build.Project,
		Id:        build.ID,
	})
	if err != nil {
		return "", false, err
	}
	if b.FinishTime == nil {
		return "", false, nil
	}
	if b.Status != cloudbuildpb.Build_SUCCESS {
		return "", false, fmt.Errorf("build %q failed: %v", build.ID, b.FailureInfo)
	}
	return b.StatusDetail, true, nil
}

func (c *RealCloudBuildClient) ResultFS(ctx context.Context, build CloudBuild) (fs.FS, error) {
	return gcsfs.FromURL(ctx, c.StorageClient, build.ResultURL)
}
