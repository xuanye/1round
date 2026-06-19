package realtime

type room struct {
	clients map[*Client]struct{}
}

func newRoom() *room {
	return &room{clients: map[*Client]struct{}{}}
}
