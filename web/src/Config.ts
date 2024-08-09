/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

export default {
  /** Limit Pull Request description size */
  PULL_REQUEST_DESCRIPTION_SIZE_LIMIT: 65_536,

  /**
   * Browser has performance issue rendering text with long line. Use a max line size
   * to tell user to cut long line into multiple smaller ones.
   */
  MAX_TEXT_LINE_SIZE_LIMIT: 5_000,

  /** Limit for a diff to be considered as a large diff */
  PULL_REQUEST_LARGE_DIFF_CHANGES_LIMIT: 1_000,

  /** Number of diffs are grouped into a single block for rendering optimization purposes */
  PULL_REQUEST_DIFF_RENDERING_BLOCK_SIZE: 10,

  /** Detection margin for on-screen / off-screen rendering optimization. In pixels.  */
  IN_VIEWPORT_DETECTION_MARGIN: 256_000,

  /** Limit for the secret input in bytes */
  SECRET_LIMIT_IN_BYTES: 5_242_880
} as const
