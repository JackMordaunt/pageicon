package pageicon

type bySize []*Icon

func (icons bySize) Len() int {
	return len(icons)
}
func (icons bySize) Swap(i, j int) {
	icons[i], icons[j] = icons[j], icons[i]
}
func (icons bySize) Less(i, j int) bool {
	return icons[i].Size > icons[j].Size
}
