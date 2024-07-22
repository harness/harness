import React from 'react'
import { NavArrowDown } from '@harnessio/icons-noir'
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from '@harnessio/canary'

interface CompanyProps {
  name: string
  avatar: React.ReactElement<SVGSVGElement>
}

const NavCompanyBadge: React.FC<CompanyProps> = ({ avatar, name }) => {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="select-none outline-none">
        <div className="grid grid-cols-[auto_1fr_auto] w-full items-center gap-2.5 justify-items-start">
          <div className="navbar-company-avatar">{avatar}</div>
          <p className="text-[15px] text-primary truncate" aria-label={name}>
            {name || 'No name'}
          </p>
          <NavArrowDown className="h-3 w-3 shrink-0 text-primary transition-transform" />
        </div>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[180px] mt-3.5 p-2.5">
        <p className="text-xs text-foreground">Company settings...</p>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export default NavCompanyBadge
