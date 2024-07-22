import React from 'react'

const Topbar = {
  Root: function Root({ children }: { children: React.ReactNode }) {
    return (
      <div className="w-full grid grid-cols-[1fr_auto] px-5 h-14 gap-6 border-b items-center text-sm font-regular">
        {children}
      </div>
    )
  },

  Left: React.memo(function Header({ children }: { children: React.ReactNode }) {
    return <div className="flex order-1 gap-3">{children}</div>
  }),

  Right: React.memo(function Header({ children }: { children: React.ReactNode }) {
    return <div className="flex order-2 gap-3">{children}</div>
  })
}

export default Topbar
