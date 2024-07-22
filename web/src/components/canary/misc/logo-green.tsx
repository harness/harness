import React from 'react'
import logoBlur from 'images/logo-green-blur.svg?url'
import logo from 'images/gitness-logo-green.svg?url'

export default function Logo() {
  return (
    <div className="w-16 h-16 relative">
      <img
        src={logoBlur}
        className="absolute bg-center opacity-[17%] max-w-[254px] w-[254px] h-[254px] -left-[calc(254px-64px)/2] -top-[calc(254px-64px)/2]"
      />
      <img
        src={logo}
        className="absolute bg-contain max-w-24 w-24 h-24 -left-[calc(96px-64px)/2] -top-[calc(96px-64px)/2]"
      />
    </div>
  )
}
