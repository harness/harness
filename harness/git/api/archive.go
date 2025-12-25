// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
)

type ArchiveFormat string

const (
	ArchiveFormatTar   ArchiveFormat = "tar"
	ArchiveFormatZip   ArchiveFormat = "zip"
	ArchiveFormatTarGz ArchiveFormat = "tar.gz"
	ArchiveFormatTgz   ArchiveFormat = "tgz"
)

var ArchiveFormats = []ArchiveFormat{
	ArchiveFormatTar,
	ArchiveFormatZip,
	ArchiveFormatTarGz,
	ArchiveFormatTgz,
}

func ParseArchiveFormat(format string) (ArchiveFormat, error) {
	switch format {
	case "tar":
		return ArchiveFormatTar, nil
	case "zip":
		return ArchiveFormatZip, nil
	case "tar.gz":
		return ArchiveFormatTarGz, nil
	case "tgz":
		return ArchiveFormatTgz, nil
	default:
		return "", errors.InvalidArgument("failed to parse file format '%s' is invalid", format)
	}
}

func (f ArchiveFormat) Validate() error {
	switch f {
	case ArchiveFormatTar, ArchiveFormatZip, ArchiveFormatTarGz, ArchiveFormatTgz:
		return nil
	default:
		return errors.InvalidArgument("git archive flag format '%s' is invalid", f)
	}
}

type ArchiveAttribute string

const (
	ArchiveAttributeExportIgnore ArchiveAttribute = "export-ignore"
	ArchiveAttributeExportSubst  ArchiveAttribute = "export-subst"
)

func (f ArchiveAttribute) Validate() error {
	switch f {
	case ArchiveAttributeExportIgnore, ArchiveAttributeExportSubst:
		return nil
	default:
		return fmt.Errorf("git archive flag worktree-attributes '%s' is invalid", f)
	}
}

type ArchiveParams struct {
	// Format of the resulting archive. Possible values are tar, zip, tar.gz, tgz,
	// and any format defined using the configuration option tar.<format>.command
	// If --format is not given, and the output file is specified, the format is inferred
	// from the filename if possible (e.g. writing to foo.zip makes the output to be in the zip format),
	// Otherwise the output format is tar.
	Format ArchiveFormat

	// Prefix prepend <prefix>/ to paths in the archive. Can be repeated; its rightmost value is used
	// for all tracked files.
	Prefix string

	// Write the archive to <file> instead of stdout.
	File string

	// export-ignore
	// Files and directories with the attribute export-ignore won’t be added to archive files.
	// See gitattributes[5] for details.
	//
	// export-subst
	// If the attribute export-subst is set for a file then Git will expand several placeholders
	// when adding this file to an archive. See gitattributes[5] for details.
	Attributes ArchiveAttribute

	// Set modification time of archive entries. Without this option the committer time is
	// used if <tree-ish> is a commit or tag, and the current time if it is a tree.
	Time *time.Time

	// Compression is level used for tar.gz and zip packers.
	Compression *int

	// The tree or commit to produce an archive for.
	Treeish string

	// Paths is optional parameter, all files and subdirectories of the
	// current working directory are included in the archive, if one or more paths
	// are specified, only these are included.
	Paths []string
}

func (p *ArchiveParams) Validate() error {
	if p.Treeish == "" {
		return errors.InvalidArgument("treeish field cannot be empty")
	}
	//nolint:revive
	if err := p.Format.Validate(); err != nil {
		return err
	}
	return nil
}

func (g *Git) Archive(ctx context.Context, repoPath string, params ArchiveParams, w io.Writer) error {
	if err := params.Validate(); err != nil {
		return err
	}
	cmd := command.New("archive",
		command.WithArg(params.Treeish),
	)

	format := ArchiveFormatTar
	if params.Format != "" {
		format = params.Format
	}
	cmd.Add(command.WithFlag("--format", string(format)))

	if params.Prefix != "" {
		prefix := params.Prefix
		if !strings.HasSuffix(params.Prefix, "/") {
			prefix += "/"
		}
		cmd.Add(command.WithFlag("--prefix", prefix))
	}

	if params.File != "" {
		cmd.Add(command.WithFlag("--output", params.File))
	}

	if params.Attributes != "" {
		if err := params.Attributes.Validate(); err != nil {
			return err
		}
		cmd.Add(command.WithFlag("--worktree-attributes", string(params.Attributes)))
	}

	if params.Time != nil {
		cmd.Add(command.WithFlag("--mtime", fmt.Sprintf("%q", params.Time.Format(time.DateTime))))
	}

	if params.Compression != nil {
		switch params.Format {
		case ArchiveFormatZip:
			// zip accepts values digit 0-9
			if *params.Compression < 0 || *params.Compression > 9 {
				return errors.InvalidArgument("compression level argument '%d' not supported for format 'zip'",
					*params.Compression)
			}
			cmd.Add(command.WithArg(fmt.Sprintf("-%d", *params.Compression)))
		case ArchiveFormatTarGz, ArchiveFormatTgz:
			// tar.gz accepts number
			cmd.Add(command.WithArg(fmt.Sprintf("-%d", *params.Compression)))
		case ArchiveFormatTar:
			// not usable for tar
		}
	}

	cmd.Add(command.WithArg(params.Paths...))

	if err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(w)); err != nil {
		return fmt.Errorf("failed to archive repository: %w", err)
	}

	return nil
}
