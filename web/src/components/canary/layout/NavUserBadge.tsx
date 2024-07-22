import React from 'react'
import userAvatar from 'images/user-avatar.svg?url'

export default function NavUserBadge() {
  return (
    <div className="grid grid-rows-2 grid-cols-[auto_1fr] gap-x-3 items-center justify-start cursor-pointer">
      <img src={userAvatar} className="col-start-1 row-span-2" />
      <p className="col-start-2 row-start-1 text-xs text-primary">Steven M.</p>
      <p className="col-start-2 row-start-2 text-xs font-light">Admin</p>
    </div>
  )
}
