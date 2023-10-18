#!/usr/bin/env node

import fs from 'fs'
import path from 'path'

fs.rmSync(path.join('.', 'dist'), { recursive: true, force: true })
