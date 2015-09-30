package raven

type Writer struct {
	Client *Client
	Level  Severity
	Logger string // Logger name reported to Sentry
}

// Write formats the byte slice p into a string, and sends a message to
// Sentry at the severity level indicated by the Writer w.
func (w *Writer) Write(p []byte) (int, error) {
	message := string(p)

	packet := NewPacket(message, &Message{message, nil})
	packet.Level = w.Level
	packet.Logger = w.Logger
	w.Client.Capture(packet, nil)

	return len(p), nil
}
