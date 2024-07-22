import React from 'react'

const Container = {
  Root: function Root({ children }: { children: React.ReactNode }) {
    return <div className="grid grid-cols-[auto_1fr] w-screen h-screen bg-[#0F0F11]">{children}</div>
  },

  Sidebar: React.memo(function Header({ children }: { children: React.ReactNode }) {
    return <div className="flex h-screen">{children}</div>
  }),

  Main: function Content({ children }: { children: React.ReactNode }) {
    return <div className="grid grid-rows-[auto_1fr_auto] col-start-2 w-full h-full">{children}</div>
  },

  Topbar: function Content({ children }: { children: React.ReactNode }) {
    return <div className="flex border-b">{children}</div>
  },

  Content: function Content({ children }: { children: React.ReactNode }) {
    return <div className="flex w-full h-full overflow-y-auto">{children}</div>
  },

  CenteredContent: function CenteredContent({ children }: { children: React.ReactNode }) {
    return (
      <div className="flex row-start-2 place-content-center items-center w-full h-full overflow-y-auto">{children}</div>
    )
  },

  Bottombar: function Content({ children }: { children: React.ReactNode }) {
    return <div className="flex border-t">{children}</div>
  }
}

export default Container
