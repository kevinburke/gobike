package geo

func Rewind(coords [][][][]float64) {
	for k, _ := range coords {
		for j, _ := range coords[k] {
			for i := len(coords[k][j])/2 - 1; i >= 0; i-- {
				opp := len(coords[k][j]) - 1 - i
				coords[k][j][i], coords[k][j][opp] = coords[k][j][opp], coords[k][j][i]
			}
		}
	}
}
