import React, { ReactNode } from 'react'

interface footerStrapProps {
  children: ReactNode
}

export default function FooterStrap({ children }: footerStrapProps) {
  return <div className="fixed z-10 bottom-0 left-0 right-0 py-8 flex justify-center bg-background">{children}</div>
}
