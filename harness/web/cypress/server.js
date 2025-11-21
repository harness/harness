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

const express = require('express')
const fs = require('fs')
const app = express()
const path = require('path')
const PORT = 8080

function checkFileExistsSync(filepath) {
  let flag = true
  try {
    fs.accessSync(filepath, fs.constants.F_OK)
  } catch (e) {
    flag = false
  }
  return flag
}

function getFile(req) {
  const url = req.url.split('?')[0].replace(/^\/gateway/, '')

  return path.resolve(__dirname, `fixtures${url}${req.method === 'POST' ? '.post' : ''}.json`)
}

app.all('*', (req, res) => {
  if (req.get('Content-Type') === 'text/plain') {
    res.setHeader('Content-Type', 'text/plain')
  } else {
    res.setHeader('Content-Type', 'application/json')
  }
  const file = getFile(req)
  if (['POST', 'GET', 'PUT'].indexOf(req.method) > -1 && checkFileExistsSync(file)) {
    res.end(fs.readFileSync(file))
  } else {
    res.end(
      JSON.stringify({
        correlationId: '',
        data: {},
        status: 'SUCCESS'
      })
    )
  }
})

app.listen(PORT, () => {
  console.log(`⚡️[server]: Server is running at http://localhost:${PORT}`)
})
