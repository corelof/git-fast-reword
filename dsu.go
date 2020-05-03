package main

type dsu struct {
	p    map[string]string
	comp int
}

func newDsu() *dsu {
	return &dsu{
		p:    make(map[string]string),
		comp: 0,
	}
}

func (d *dsu) addNode(node string) {
	if _, ok := d.p[node]; ok {
		return
	}
	d.p[node] = node
	d.comp++
}

func (d *dsu) fp(a string) string {
	if d.p[a] == a {
		return a
	}
	d.p[a] = d.fp(d.p[a])
	return d.p[a]
}

func (d *dsu) connected(a string, b string) bool {
	if _, ok := d.p[a]; !ok {
		return false
	}
	if _, ok := d.p[b]; !ok {
		return false
	}
	return d.fp(a) == d.fp(b)
}

func (d *dsu) connect(a string, b string) {
	if _, ok := d.p[a]; !ok {
		return
	}
	if _, ok := d.p[b]; !ok {
		return
	}
	if !d.connected(a, b) {
		a = d.fp(a)
		b = d.fp(b)
		d.p[b] = a
		d.comp--
	}
}
