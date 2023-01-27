package prometheus

type Builder struct {
	client *Client
}

func (prometheus Client) NewBuilder() *Builder {
	return &Builder{client: &prometheus}
}
