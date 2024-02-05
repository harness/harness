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

//
// This file contains global settings for the whole application. Do not store page or
// component specific consts here.
//

export const PULL_REQUEST_DESCRIPTION_SIZE_LIMIT = 65_536

// Browser has performance issue rendering text with long line. Use a max line size
// to tell user to cut long line into multiple smaller ones.
export const MAX_TEXT_LINE_SIZE_LIMIT = 5_000
