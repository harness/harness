# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Top-level presubmit script.

See https://dev.chromium.org/developers/how-tos/depottools/presubmit-scripts for
details on the presubmit API built into depot_tools.
"""

import os
import sys


def PreCommitGo(input_api, output_api, pcg_mode):
  """Run go-specific checks via pre-commit-go (pcg) if it's in PATH."""
  if input_api.is_committing:
    error_type = output_api.PresubmitError
  else:
    error_type = output_api.PresubmitPromptWarning

  exe = 'pcg.exe' if sys.platform == 'win32' else 'pcg'
  pcg = None
  for p in os.environ['PATH'].split(os.pathsep):
    pcg = os.path.join(p, exe)
    if os.access(pcg, os.X_OK):
      break
  else:
    return [
      error_type(
        'pre-commit-go executable (pcg) could not be found in PATH. All Go '
        'checks are skipped. See https://github.com/maruel/pre-commit-go.')
    ]

  cmd = [pcg, 'run', '-m', ','.join(pcg_mode)]
  if input_api.verbose:
    cmd.append('-v')
  # pcg can figure out what files to check on its own based on upstream ref,
  # but on PRESUBMIT try builder upsteram isn't set, and it's just 1 commit.
  if os.getenv('PRESUBMIT_BUILDER', ''):
    cmd.extend(['-r', 'HEAD~1'])
  return input_api.RunTests([
    input_api.Command(
      name='pre-commit-go: %s' % ', '.join(pcg_mode),
      cmd=cmd,
      kwargs={},
      message=error_type),
  ])


def header(input_api):
  """Returns the expected license header regexp for this project."""
  current_year = int(input_api.time.strftime('%Y'))
  allowed_years = (str(s) for s in reversed(xrange(2011, current_year + 1)))
  years_re = '(' + '|'.join(allowed_years) + ')'
  license_header = (
    r'.*? Copyright %(year)s The Chromium Authors\. '
    r'All rights reserved\.\n'
    r'.*? Use of this source code is governed by a BSD-style license '
    r'that can be\n'
    r'.*? found in the LICENSE file\.(?: \*/)?\n'
  ) % {
    'year': years_re,
  }
  return license_header


def source_file_filter(input_api):
  """Returns filter that selects source code files only."""
  bl = list(input_api.DEFAULT_BLACK_LIST) + [
    r'.+\.pb\.go$',
    r'.+_string\.go$',
  ]
  wl = list(input_api.DEFAULT_WHITE_LIST) + [
    r'.+\.go$',
  ]
  return lambda x: input_api.FilterSourceFile(x, white_list=wl, black_list=bl)


def CommonChecks(input_api, output_api):
  results = []
  results.extend(
    input_api.canned_checks.CheckChangeHasNoStrayWhitespace(
      input_api, output_api,
      source_file_filter=source_file_filter(input_api)))
  results.extend(
    input_api.canned_checks.CheckLicense(
      input_api, output_api, header(input_api),
      source_file_filter=source_file_filter(input_api)))
  return results


def CheckChangeOnUpload(input_api, output_api):
  results = CommonChecks(input_api, output_api)
  results.extend(PreCommitGo(input_api, output_api, ['lint', 'pre-commit']))
  return results


def CheckChangeOnCommit(input_api, output_api):
  results = CommonChecks(input_api, output_api)
  results.extend(input_api.canned_checks.CheckChangeHasDescription(
      input_api, output_api))
  results.extend(input_api.canned_checks.CheckDoNotSubmitInDescription(
      input_api, output_api))
  results.extend(input_api.canned_checks.CheckDoNotSubmitInFiles(
      input_api, output_api))
  results.extend(PreCommitGo(
      input_api, output_api, ['continuous-integration']))
  return results
