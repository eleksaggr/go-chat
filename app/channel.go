package app

type Channel struct {
	Name    string
	Members []User
}

func NewChannel(name string) *Channel {
	return &Channel{Name: name}
}

func (c *Channel) Add(user User) {
  c.Members = append(c.Members, user)
}

func (c *Channel) Remove(id uint32) {
	var pos int
	for i, user := range c.Members {
		if(user.Id == id) {
			pos = i
			break
		}
	}

	c.Members = append(c.Members[:pos], c.Members[pos+1:]...)
}
