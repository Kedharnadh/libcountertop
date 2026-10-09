package widgets

import "github.com/ygelfand/libcountertop/pkg/tessera"

const routeMost = 8

func (v *View) showing(l tessera.Layout) int {
	v.mu.Lock()
	defer v.mu.Unlock()
	if l.Revision != v.revision {
		v.revision, v.page = l.Revision, l.Home
		for i, p := range l.Pages {
			if p.ID == v.shown && v.shown != "" {
				v.page = i
			}
		}
	}
	if v.rested {
		v.rested = false
		if v.o.HomeAfterIdle != nil && v.o.HomeAfterIdle() {
			v.page, v.route, v.card = l.Home, nil, ""
		}
	}
	if v.page < 0 || v.page >= len(l.Pages) {
		v.page, v.route = l.Home, nil
	}
	v.shown = l.Pages[v.page].ID
	return v.page
}

func (v *View) follow(s *tessera.Screen, l tessera.Layout) {
	a := s.Asked()
	v.mu.Lock()
	defer v.mu.Unlock()
	if a.Ask == v.asked {
		return
	}
	v.asked = a.Ask
	if a.Index < 0 || a.Index >= len(l.Pages) {
		return
	}
	v.revision = l.Revision
	v.page, v.route, v.card, v.camera, v.sub = a.Index, nil, "", "", sub{}
	v.lift, v.code = 0, code{}
	v.shown = l.Pages[a.Index].ID
}

func (v *View) home(l tessera.Layout) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.page, v.route, v.card, v.camera = l.Home, nil, "", ""
	v.shown = l.Pages[v.page].ID
}

func (v *View) open(l tessera.Layout, to int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if to < 0 || to >= len(l.Pages) || to == v.page {
		return
	}
	v.route = append(v.route, l.Pages[v.page].ID)
	if len(v.route) > routeMost {
		v.route = v.route[len(v.route)-routeMost:]
	}
	v.page = to
	v.shown = l.Pages[to].ID
}

func (v *View) back(l tessera.Layout) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for len(v.route) > 0 {
		id := v.route[len(v.route)-1]
		v.route = v.route[:len(v.route)-1]
		for i, p := range l.Pages {
			if p.ID == id && i != v.page {
				v.page, v.shown = i, p.ID
				return
			}
		}
	}
	v.page = l.Home
	v.shown = l.Pages[l.Home].ID
}

func pages(l tessera.Layout) []int {
	var out []int
	for i, p := range l.Pages {
		if !p.Excluded {
			out = append(out, i)
		}
	}
	return out
}
