package mincer

import (
	"sort"
	"time"

	"golang.org/x/exp/rand"
)

type Group struct {
	start int
	end   int
}

type Mincer struct {
	MinRate          int // 0 ~ 100
	MaxRate          int // 0 ~ 100
	ChunkSize        int
	MinChunkPerGroup int
	MaxChunkPerGroup int
	r                *rand.Rand
	maxGroupSize     int
}

func (m *Mincer) Init(seed *uint64) error {
	if seed == nil {
		s := uint64(time.Now().UnixNano())
		seed = &s
	}
	r := rand.New(rand.NewSource(*seed))
	m.r = r

	m.maxGroupSize = m.MaxChunkPerGroup * m.ChunkSize
	return nil
}

// return an int in [min, max]
func (m *Mincer) randInt(min int, max int) int {
	return m.r.Intn(max-min+1) + min
}

func (m *Mincer) getChunkGroups(size int, numGroup int) []Group {
	nums := make([]int, numGroup)
	for i := 0; i < numGroup; i++ {
		nums[i] = m.randInt(0, size)
	}
	sort.Ints(nums)
	groupIdxList := make([]int, 0)
	for i := 0; i < numGroup; i++ {
		// if 2 numbers are too close, ignore the 2nd number
		if i > 0 && nums[i-1]+m.maxGroupSize >= nums[i] {
			continue
		}
		groupIdxList = append(groupIdxList, nums[i])
	}
	numGroup = len(groupIdxList)
	// fill in details
	groups := make([]Group, 0)
	for i := 0; i < numGroup; i++ {
		gIdx := groupIdxList[i]
		if i == 0 && gIdx != 0 {
			groups = append(groups, Group{start: 0, end: gIdx - 1})
		}
		var prevGroupEnd int
		if len(groups) > 0 {
			prevGroupEnd = groups[len(groups)-1].end
		} else {
			prevGroupEnd = -1
		}
		numChunk := m.randInt(m.MinChunkPerGroup, m.MaxChunkPerGroup)
		subGroups := make([]Group, 0)
		left := prevGroupEnd + 1
		for j := 0; j < numChunk; j++ {
			g := Group{
				start: left,
				end:   left + m.ChunkSize,
			}
			subGroups = append(subGroups, g)
			if g.end > size-1 {
				subGroups[j].end = size - 1
				break
			}
			left = g.end + 1
		}
		groups = append(groups, subGroups...)
		lastGroup := groups[len(groups)-1]
		if lastGroup.end == size-1 {
			break
		}
		if i == numGroup-1 && lastGroup.end < size-1 {
			groups = append(
				groups,
				Group{start: lastGroup.end + 1, end: size - 1},
			)
		}
	}
	return groups
}

func (m *Mincer) Run(data []byte) [][]byte {
	size := m.randInt(m.MinRate, m.MaxRate) * len(data) / 100
	if size < 1 {
		size = 1
	}
	avgGroupSize := (m.MinChunkPerGroup + m.MaxChunkPerGroup) / 2
	numGroup := size / m.ChunkSize / avgGroupSize
	if numGroup < 1 {
		numGroup = 1
	}
	groups := m.getChunkGroups(len(data), numGroup)

	rst := make([][]byte, len(groups))
	for i := 0; i < len(groups); i++ {
		g := groups[i]
		rst[i] = data[g.start : g.end+1]
	}
	return rst
}
