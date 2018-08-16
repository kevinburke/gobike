package geo

func Rewind(coords [][][][]float64) {
	for k := range coords {
		for j := range coords[k] {
			for i := len(coords[k][j])/2 - 1; i >= 0; i-- {
				opp := len(coords[k][j]) - 1 - i
				coords[k][j][i], coords[k][j][opp] = coords[k][j][opp], coords[k][j][i]
			}
		}
	}
}
