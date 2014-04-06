package hipchat

type Room struct {
	// The ID of the room.
	Id int `json:"room_id"`

	// The name of the room.
	Name string

	// The current room topic.
	Topic string

	// Time of last activity (sent message) in the room in UNIX time (UTC).
	// May be 0 in rare cases when the time is unknown.
	LastActive int `json:"last_active"`

	// Time the room was created in UNIX time (UTC).
	Created int

	// Whether or not this room is archived.
	Archived bool `json:"is_archived"`

	// Whether or not this room is private.
	Private bool `json:"is_private"`

	// User ID of the room owner.
	OwnerUserId int `json:"owner_user_id"`

	// XMPP/Jabber ID of the room.
	XMPPJabberId string `json:"xmpp_jid"`
}
