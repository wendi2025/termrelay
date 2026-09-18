package service

import "sort"

// rankAccountPoolCandidatesWithStaticWeight preserves strict priority tiers:
// lower numeric priority is always exhausted before a lower-priority tier can
// receive traffic. Inside each priority tier, runtime health/load scoring gates
// the Top-K set and operator scheduling_weight only distributes traffic among
// those viable peers.
func rankAccountPoolCandidatesWithStaticWeight(candidates []accountPoolCandidateScore, topK int, seed string) []accountPoolCandidateScore {
	if len(candidates) == 0 {
		return nil
	}

	ranked := append([]accountPoolCandidateScore(nil), candidates...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Account.Priority != ranked[j].Account.Priority {
			return ranked[i].Account.Priority < ranked[j].Account.Priority
		}
		if ranked[i].Score != ranked[j].Score {
			return ranked[i].Score > ranked[j].Score
		}
		return ranked[i].Account.ID < ranked[j].Account.ID
	})

	out := make([]accountPoolCandidateScore, 0, len(ranked))
	for start := 0; start < len(ranked); {
		end := start + 1
		priority := ranked[start].Account.Priority
		for end < len(ranked) && ranked[end].Account.Priority == priority {
			end++
		}

		tier := append([]accountPoolCandidateScore(nil), ranked[start:end]...)
		tierTopK := topK
		if tierTopK <= 0 || tierTopK > len(tier) {
			tierTopK = len(tier)
		}

		// Health/load score decides who is allowed into Top-K. Static weight only
		// affects traffic sharing inside that already-vetted set.
		top := append([]accountPoolCandidateScore(nil), tier[:tierTopK]...)
		sort.SliceStable(top, func(i, j int) bool {
			return accountPoolWeightedRendezvous(seed, top[i]) > accountPoolWeightedRendezvous(seed, top[j])
		})
		out = append(out, top...)
		out = append(out, tier[tierTopK:]...)
		start = end
	}
	return out
}

func accountPoolWeightedRendezvous(seed string, candidate accountPoolCandidateScore) float64 {
	base := accountPoolRendezvous(seed, candidate)
	if candidate.Account == nil {
		return base
	}
	return base * candidate.Account.SchedulingWeight()
}
