package delta

import "github.com/mezotov/netdiscover/internal/model"

func Compute(prev, curr []*model.Device) Delta {
	prevMap := map[string]*model.Device{}
	currMap := map[string]*model.Device{}

	for _, d := range prev {
		prevMap[d.ID] = d
	}
	for _, d := range curr {
		currMap[d.ID] = d
	}

	var out Delta

	for id, d := range currMap {
		if _, ok := prevMap[id]; !ok {
			out.New = append(out.New, d)
		} else {
			out.Updated = append(out.Updated, d)
		}
	}

	for id, d := range prevMap {
		if _, ok := currMap[id]; !ok {
			out.Removed = append(out.Removed, d)
		}
	}

	return out
}
