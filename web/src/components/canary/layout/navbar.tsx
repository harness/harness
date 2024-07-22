import React from 'react'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger, cn } from '@harnessio/canary'

const Navbar = {
  Root: function Root({ children }: { children: React.ReactNode }) {
    return (
      <div className="select-none grid grid-rows-[auto_1fr_auto] w-[220px] h-screen overflow-y-auto border-r text-sm text-[#AEAEB7] bg-[#070709]">
        {children}
      </div>
    )
  },

  Header: React.memo(function Header({ children }: { children: React.ReactNode }) {
    return <div className="px-5 h-[57px] items-center grid">{children}</div>
  }),

  Content: function Content({ children }: { children: React.ReactNode }) {
    return <div className="grid content-start">{children}</div>
  },

  Group: function Group({ children, topBorder }: { children: React.ReactNode; topBorder?: boolean }) {
    return (
      <div
        className={cn('p-5 py-3.5 flex flex-col gap-1.5', {
          'border-t': topBorder
        })}>
        {children}
      </div>
    )
  },

  AccordionGroup: function AccordionGroup({ title, children }: { title: string; children: React.ReactNode }) {
    return (
      <div className="p-5 py-0.5 border-t">
        <Accordion type="single" collapsible defaultValue="item-1">
          <AccordionItem value="item-1" className="border-none">
            <AccordionTrigger className="group">
              <p className="text-xs text-[#60606C] font-normal group-hover:text-primary ease-in-out duration-150">
                {title}
              </p>
            </AccordionTrigger>
            <AccordionContent className="flex flex-col gap-1.5">{children}</AccordionContent>
          </AccordionItem>
        </Accordion>
      </div>
    )
  },

  Item: React.memo(
    ({
      icon,
      text,
      active
    }: {
      icon: React.ReactElement<SVGSVGElement>
      text: string
      active?: boolean
      onClick?: () => void
    }) => {
      return (
        <div
          className={cn('navbar-item flex gap-2.5 items-center cursor-pointer group select-none py-1.5', {
            'active text-primary': active
          })}>
          <div className="flex items-center">{icon}</div>
          <p
            className={cn('-tracking-[2%] ease-in-out duration-150 truncate', {
              'text-primary': active,
              'group-hover:text-primary': !active
            })}>
            {text}
          </p>
        </div>
      )
    }
  ),

  Footer: React.memo(function Footer({ children }: { children: React.ReactNode }) {
    return <div className="grid px-5 h-[76px] items-center border-t">{children}</div>
  })
}

export default Navbar
