import React from 'react'
import { Link } from 'react-router-dom'
import css from './GitRefLink.module.scss'

export const GitRefLink: React.FC<{ text: string; url: string }> = ({ text, url }) => (
  <Link to={url} className={css.link}>
    {text}
  </Link>
)
