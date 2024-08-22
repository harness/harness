/*
 * Copyright 2024 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import express from 'express'
import { exec } from 'child_process'
import path from 'path'
import { v4 as uuidv4 } from 'uuid'

const app = express()
const port = 3001

interface ScriptStatus {
  status: 'pending' | 'running' | 'completed' | 'error'
  output: string
  error: string
}

const scriptStatuses: Record<string, ScriptStatus> = {}

app.get('/execute-script', (req, res) => {
  const { script, params = '' } = req.query
  const scriptId = uuidv4()
  scriptStatuses[scriptId] = { status: 'pending', output: '', error: '' }
  const scriptPath = path.join(process.cwd(), `scripts/${script}`)
  const command = `sh ${scriptPath} ${params}`

  console.log(scriptPath, params, command)
  const child = exec(command)

  scriptStatuses[scriptId].status = 'running'

  child.stdout.on('data', data => {
    scriptStatuses[scriptId].output += data
  })

  child.stderr.on('data', data => {
    scriptStatuses[scriptId].error += data
  })

  child.on('close', code => {
    if (code === 0) {
      scriptStatuses[scriptId].status = 'completed'
    } else {
      scriptStatuses[scriptId].status = 'error'
    }
  })

  res.send({ scriptId })
})

app.get('/script-status/:id', (req, res) => {
  const { id } = req.params
  const status = scriptStatuses[id]

  if (!status) {
    res.status(404).send('Script ID not found')
    return
  }

  res.send(status)
})

app.listen(port, () => {
  console.log(`Server is running at http://localhost:${port}`)
})
