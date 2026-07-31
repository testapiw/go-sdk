package transport

type Context map[string]any

func NewContext() Context {
	return make(Context)
}

func (c Context) Set(key string, value any) {
	c[key] = value
}

func (c Context) Get(key string) (any, bool) {
	v, ok := c[key]
	return v, ok
}

func (c Context) MustGet(key string) any {
	return c[key]
}