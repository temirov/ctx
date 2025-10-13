package callchain

import "sort"

func collectReachable(start string, adjacency map[string]map[string]struct{}, maximumDepth int) []string {
	if maximumDepth <= 0 {
		return nil
	}
	visited := map[string]struct{}{start: {}}
	frontier := []string{start}
	var collected []string
	for depth := 0; depth < maximumDepth; depth++ {
		var next []string
		for _, node := range frontier {
			neighbors := adjacency[node]
			for neighbor := range neighbors {
				if _, seen := visited[neighbor]; seen {
					continue
				}
				visited[neighbor] = struct{}{}
				collected = append(collected, neighbor)
				next = append(next, neighbor)
			}
		}
		if len(next) == 0 {
			break
		}
		frontier = next
	}
	sort.Strings(collected)
	return collected
}
