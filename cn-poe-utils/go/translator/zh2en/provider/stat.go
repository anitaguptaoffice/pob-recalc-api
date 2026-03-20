package provider

import (
	"slices"
	"strings"

	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
	"github.com/cn-poe-community/cn-poe-utils/go/translator/zh2en/util"
)

type MultilineStatGroup struct {
	MaxLineSize int
	Stats       []MultilineStat
}

type MultilineStat struct {
	LineSize int
	Stat     *poe.Stat
}

type StatProvider struct {
	zhSkeletonIdx          map[string][]*poe.Stat
	firstLineZhSkeletonIdx map[string]*MultilineStatGroup
	referenceStats         []*poe.Stat
	multilineRefStats      []MultilineStat
}

func NewStatProvider(data *poe.Data) *StatProvider {
	zhSkeletonIdx := make(map[string][]*poe.Stat)
	firstLineZhSkeletonIdx := make(map[string]*MultilineStatGroup)
	referenceStats := make([]*poe.Stat, 0)
	multilineRefStats := make([]MultilineStat, 0)

	for i, item := range data.Stats {
		if item.Refs != nil {
			referenceStats = append(referenceStats, &data.Stats[i])
			if strings.Contains(item.Zh, util.LINE_SEPARATOR) {
				multilineRefStats = append(multilineRefStats, MultilineStat{
					LineSize: strings.Count(item.Zh, util.LINE_SEPARATOR) + 1,
					Stat:     &data.Stats[i],
				})
			}
			continue
		}
		skeleton := util.GetTextSkeleton(item.Zh)
		zhSkeletonIdx[skeleton] = append(zhSkeletonIdx[skeleton], &data.Stats[i])

		if strings.Contains(item.Zh, util.LINE_SEPARATOR) {
			lines := strings.Split(item.Zh, util.LINE_SEPARATOR)
			lineCount := len(lines)
			firstLine := lines[0]
			firstLineSkeleton := util.GetTextSkeleton(firstLine)

			if _, ok := firstLineZhSkeletonIdx[firstLineSkeleton]; !ok {
				firstLineZhSkeletonIdx[firstLineSkeleton] = &MultilineStatGroup{
					MaxLineSize: lineCount,
					Stats:       []MultilineStat{},
				}
			}
			group := firstLineZhSkeletonIdx[firstLineSkeleton]
			group.Stats = append(group.Stats, MultilineStat{
				LineSize: lineCount,
				Stat:     &data.Stats[i],
			})

			if group.MaxLineSize < lineCount {
				group.MaxLineSize = lineCount
			}

			for _, group := range firstLineZhSkeletonIdx {
				if len(group.Stats) > 0 {
					// // 按照行数从多到少排序，这样匹配时首先匹配更多的行
					slices.SortFunc(group.Stats, func(a, b MultilineStat) int {
						return b.LineSize - a.LineSize
					})
				}
			}
		}
	}
	return &StatProvider{
		zhSkeletonIdx:          zhSkeletonIdx,
		firstLineZhSkeletonIdx: firstLineZhSkeletonIdx,
		referenceStats:         referenceStats,
		multilineRefStats:      multilineRefStats,
	}
}

func (p *StatProvider) ProvideByZhSkeleton(skeleton string) []*poe.Stat {
	if stats, ok := p.zhSkeletonIdx[skeleton]; ok {
		return stats
	}
	return nil
}

func (p *StatProvider) ProvideByFirstLineZhSkeleton(skeleton string) *MultilineStatGroup {
	if group, ok := p.firstLineZhSkeletonIdx[skeleton]; ok {
		return group
	}
	return nil
}

func (p *StatProvider) ProvideReferenceStats() []*poe.Stat {
	return p.referenceStats
}

func (p *StatProvider) ProvideMultilineReferenceStats() []MultilineStat {
	return p.multilineRefStats
}
