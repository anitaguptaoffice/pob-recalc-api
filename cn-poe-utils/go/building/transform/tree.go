package transform

import (
	"fmt"
	"slices"
	"sort"

	"github.com/cn-poe-community/cn-poe-utils/go/api"
	"github.com/cn-poe-community/cn-poe-utils/go/building/util"
	"github.com/cn-poe-community/cn-poe-utils/go/data/pob"
)

// GetNodeIdOfExpansionSlot 获取扩展插槽的节点ID
func GetNodeIdOfExpansionSlot(seqNum int) int {
	return pob.DefaultData.Tree.JewelSlots[seqNum]
}

// GetCharacterName 获取角色名称
func GetCharacterName(num int) string {
	return pob.DefaultData.Tree.Classes[num].Name
}

// GetAscendancyName 获取升华名称
func GetAscendancyName(characterNum int, ascendancyNum int) string {
	if ascendancyNum == 0 {
		return ""
	}
	return pob.DefaultData.Tree.Classes[characterNum].Ascendancies[ascendancyNum-1].Name
}

var phreciaAscendancySet = make(map[string]struct{})

func init() {
	for _, name := range pob.DefaultData.PhreciaAscendancyMap {
		phreciaAscendancySet[name] = struct{}{}
	}
}

// IsPhreciaAscendancy 判断是否为Phrecia升华
func IsPhreciaAscendancy(name string) bool {
	_, ok := phreciaAscendancySet[name]
	return ok
}

// ClusterJewelSize 星团珠宝大小
type ClusterJewelSize string

const (
	ClusterJewelSizeLarge  ClusterJewelSize = "Large Cluster Jewel"
	ClusterJewelSizeMedium ClusterJewelSize = "Medium Cluster Jewel"
	ClusterJewelSizeSmall  ClusterJewelSize = "Small Cluster Jewel"
)

func clusterJewelSize(jewelType string) *ClusterJewelSize {
	switch jewelType {
	case "JewelPassiveTreeExpansionLarge":
		s := ClusterJewelSizeLarge
		return &s
	case "JewelPassiveTreeExpansionMedium":
		s := ClusterJewelSizeMedium
		return &s
	case "JewelPassiveTreeExpansionSmall":
		s := ClusterJewelSizeSmall
		return &s
	}
	return nil
}

// GetEnabledNodeIdsOfJewels 返回所有星团上点亮的node的nodeId
func GetEnabledNodeIdsOfJewels(passiveSkills *api.GetPassiveSkillsResult) []int {
	hashEx := passiveSkills.HashesEx
	jewelData := passiveSkills.JewelData
	items := passiveSkills.Items

	// 获取所有jewel，并按照从大到小进行排序
	jewelList := getSortedClusterJewels(jewelData, items)

	hashExSet := make(map[int]struct{})
	for _, h := range hashEx {
		hashExSet[h] = struct{}{}
	}

	// 使用proxy作为key，关联所在socket的ExpansionJewel
	// id是socket所在星团的POB内部实现细节，供子星团使用
	socketExpansionJewels := make(map[int]*struct {
		ID int
		EJ *api.ExpansionJewel
	})

	var allEnabledNodeIds []int
	// 由于API给的数据无法判断传奇小型星团珠宝的keystone是否点亮（如果使用POB原生导入，keystone是未点亮的）
	// 这里我们将其标记为可能点亮的，当我们每点亮一个节点，就从hashExSet移除关联的索引键
	// 最后我们根据hashExSet的剩余大小，来点亮相同数目的keystone，这不一定准确，但适用于99%的情况
	var allProbableNodeIds []int

	for _, jewel := range jewelList {
		seqNum := jewel.SeqNum
		size := jewel.Size

		var id *int
		var expansionJewel *api.ExpansionJewel

		// 中小型星团
		if size == ClusterJewelSizeMedium || size == ClusterJewelSizeSmall {
			group := jewel.Data.Subgraph.Groups[fmt.Sprintf("expansion_%d", seqNum)]
			proxy := util.MustAtoi(group.Proxy)
			idAndEj := socketExpansionJewels[proxy]
			// 且是（位于socket上）子星团
			if idAndEj != nil {
				id = &idAndEj.ID
				expansionJewel = idAndEj.EJ
			}
		}

		// 大型星团（必然位于slot上）或位于slot上的中小型星团
		if id == nil {
			slotNodeId := GetNodeIdOfExpansionSlot(seqNum)
			n := pob.DefaultData.Tree.Nodes[slotNodeId]
			expansionJewel = &api.ExpansionJewel{
				Size:  n.ExpansionJewel.Size,
				Index: n.ExpansionJewel.Index,
				Proxy: n.ExpansionJewel.Proxy,
			}

			if n.ExpansionJewel.Parent != nil {
				expansionJewel.Parent = *n.ExpansionJewel.Parent
			}
		}

		enabledNodeIds, probableNodeIds := getEnabledNodeIdsOfJewel(
			hashExSet,
			jewel,
			expansionJewel,
			id,
			socketExpansionJewels,
		)

		allEnabledNodeIds = append(allEnabledNodeIds, enabledNodeIds...)
		allProbableNodeIds = append(allProbableNodeIds, probableNodeIds...)
	}

	n := min(len(hashExSet), len(allProbableNodeIds))
	if n > 0 {
		allEnabledNodeIds = append(allEnabledNodeIds, allProbableNodeIds[:n]...)
	}

	return allEnabledNodeIds
}

// ClusterJewelInfo 星团珠宝信息
type ClusterJewelInfo struct {
	SeqNum int
	Item   *api.Item
	Data   *api.JewelDatum
	Size   ClusterJewelSize
}

// 获取所有星团珠宝，并按照大小降序排序
func getSortedClusterJewels(
	jewelData api.JewelData,
	items []api.Item,
) []*ClusterJewelInfo {
	itemIdx := make(map[int]*api.Item)
	for i := range items {
		item := &items[i]
		if item.X != nil {
			itemIdx[*item.X] = item
		}
	}

	var jewelList []*ClusterJewelInfo
	for i, data := range jewelData {
		seqNum := util.MustAtoi(i)
		size := clusterJewelSize(data.Type)
		if size != nil {
			jewelList = append(jewelList, &ClusterJewelInfo{
				SeqNum: seqNum,
				Item:   itemIdx[seqNum],
				Data:   &data,
				Size:   *size,
			})
		}
	}

	slices.SortFunc(jewelList, func(a, b *ClusterJewelInfo) int {
		sizeA := a.Size
		sizeB := b.Size
		// 字符串的自然序"LARGE"<"MEDIUM"<"SMALL"，与实际顺序相反
		// 这里我们需要逆序，也就是使用自然序
		if sizeA < sizeB {
			return -1
		} else if sizeA > sizeB {
			return 1
		}
		return 0
	})
	return jewelList
}

// ClusterJewelNode 星团珠宝节点
type ClusterJewelNode struct {
	ID   int // nodeId
	OIdx int // 局部序号，指使用0~11标记单个星团中的节点
}

// 返回单个星团上点亮的node的nodeId
// socketEjs用于返回填充数据，供子星团使用
func getEnabledNodeIdsOfJewel(
	hashExSet map[int]struct{},
	jewel *ClusterJewelInfo,
	expansionJewel *api.ExpansionJewel,
	id *int,
	socketEjs map[int]*struct {
		ID int
		EJ *api.ExpansionJewel
	},
) (enabledNodeIds []int, probableNodeIds []int) {
	jSize := jewel.Size
	jMeta := pob.DefaultData.ClusterJewels.Jewels[string(jSize)]

	// 算法移植自PassiveSpec.lua文件的BuildSubgraph()方法
	idGen := 0x10000
	if id != nil {
		idGen = *id
	}
	if expansionJewel.Size == 2 {
		idGen += (expansionJewel.Index << 6)
	} else if expansionJewel.Size == 1 {
		idGen += (expansionJewel.Index << 9)
	}
	nodeIdGenerator := idGen + (jMeta.SizeIndex << 4)

	// 原始的id，最终需要转换为nodeId
	var notableIds []int
	var socketIds []int
	var smallIds []int

	group := jewel.Data.Subgraph.Groups[fmt.Sprintf("expansion_%d", jewel.SeqNum)]
	originalNodeIds := make([]int, 0, len(group.Nodes))
	for _, n := range group.Nodes {
		id := util.MustAtoi(n)
		originalNodeIds = append(originalNodeIds, id)
	}
	jewelNodes := jewel.Data.Subgraph.Nodes

	// unique small cluster jewel
	isUnique := jewel.Item.Rarity != nil && *jewel.Item.Rarity == api.RarityUnique
	if len(originalNodeIds) == 0 && len(jewelNodes) == 0 && isUnique {
		probableNodeIds = append(probableNodeIds, nodeIdGenerator)
		return
	}

	for _, i := range originalNodeIds {
		node := jewelNodes[i]
		originalId := i
		if node.IsNotable != nil && *node.IsNotable {
			notableIds = append(notableIds, originalId)
		} else if node.IsJewelSocket != nil && *node.IsJewelSocket {
			socketIds = append(socketIds, originalId)
			if node.ExpansionJewel != nil {
				proxy := util.MustAtoi(node.ExpansionJewel.Proxy)
				socketEjs[proxy] = &struct {
					ID int
					EJ *api.ExpansionJewel
				}{
					ID: idGen,
					EJ: node.ExpansionJewel,
				}
			}
		} else if node.IsMastery != nil && *node.IsMastery {
			// DO NOTHING
		} else {
			smallIds = append(smallIds, originalId)
		}
	}

	nodeCount := len(notableIds) + len(socketIds) + len(smallIds)

	var pobJewelNodes []*ClusterJewelNode
	// 使用0~11索引星团中的节点
	indicies := make(map[int]*ClusterJewelNode)

	if jSize == ClusterJewelSizeLarge && len(socketIds) == 1 {
		socket := jewelNodes[socketIds[0]]
		skill := util.MustAtoi(socket.Skill)
		pobNode := &ClusterJewelNode{
			ID:   skill,
			OIdx: 6,
		}
		pobJewelNodes = append(pobJewelNodes, pobNode)
		indicies[pobNode.OIdx] = pobNode
	} else {
		for i := 0; i < len(socketIds); i++ {
			socket := jewelNodes[socketIds[i]]
			skill := util.MustAtoi(socket.Skill)
			pobNode := &ClusterJewelNode{
				ID:   skill,
				OIdx: jMeta.SocketIndicies[i],
			}
			pobJewelNodes = append(pobJewelNodes, pobNode)
			indicies[pobNode.OIdx] = pobNode
		}
	}

	var notableIndicies []int
	for _, n := range jMeta.NotableIndicies {
		if len(notableIndicies) == len(notableIds) {
			break
		}

		if jSize == ClusterJewelSizeMedium {
			if len(socketIds) == 0 && len(notableIds) == 2 {
				if n == 6 {
					n = 4
				} else if n == 10 {
					n = 8
				}
			} else if nodeCount == 4 {
				if n == 10 {
					n = 9
				} else if n == 2 {
					n = 3
				}
			}
		}
		if _, ok := indicies[n]; !ok {
			notableIndicies = append(notableIndicies, n)
		}
	}
	sort.Ints(notableIndicies)

	for i := 0; i < len(notableIndicies); i++ {
		idx := notableIndicies[i]
		pobNode := &ClusterJewelNode{
			ID:   nodeIdGenerator + idx,
			OIdx: idx,
		}
		pobJewelNodes = append(pobJewelNodes, pobNode)
		indicies[idx] = pobNode
	}

	var smallIndicies []int
	for _, n := range jMeta.SmallIndicies {
		if len(smallIndicies) == len(smallIds) {
			break
		}

		idx := n
		if jSize == ClusterJewelSizeMedium {
			if nodeCount == 5 && n == 4 {
				idx = 3
			} else if nodeCount == 4 {
				if n == 8 {
					idx = 9
				} else if n == 4 {
					idx = 3
				}
			}
		}
		if _, exists := indicies[idx]; !exists {
			smallIndicies = append(smallIndicies, idx)
		}
	}

	for i := 0; i < len(smallIndicies); i++ {
		idx := smallIndicies[i]
		pobNode := &ClusterJewelNode{
			ID:   nodeIdGenerator + idx,
			OIdx: idx,
		}
		pobJewelNodes = append(pobJewelNodes, pobNode)
		indicies[idx] = pobNode
	}

	proxy := 0
	for _, c := range expansionJewel.Proxy {
		proxy = proxy*10 + int(c-'0')
	}
	proxyNode := pob.DefaultData.Tree.Nodes[proxy]
	proxyNodeSkillsPerOrbit := pob.DefaultData.Tree.Constants.SkillsPerOrbit[proxyNode.Orbit]
	for _, node := range pobJewelNodes {
		proxyNodeOidxRelativeToClusterIndicies := translateOidx(
			proxyNode.OrbitIndex,
			proxyNodeSkillsPerOrbit,
			jMeta.TotalIndicies,
		)
		correctedNodeOidxRelativeToClusterIndicies := (node.OIdx + proxyNodeOidxRelativeToClusterIndicies) % jMeta.TotalIndicies
		correctedNodeOidxRelativeToTreeSkillsPerOrbit := translateOidx(
			correctedNodeOidxRelativeToClusterIndicies,
			jMeta.TotalIndicies,
			proxyNodeSkillsPerOrbit,
		)
		node.OIdx = correctedNodeOidxRelativeToTreeSkillsPerOrbit
		indicies[node.OIdx] = node
	}

	for _, i := range originalNodeIds {
		node := jewelNodes[i]
		originalId := i
		if _, exists := hashExSet[originalId]; exists {
			pobNode := indicies[node.OrbitIndex]
			if pobNode != nil {
				enabledNodeIds = append(enabledNodeIds, pobNode.ID)
			}
			delete(hashExSet, originalId)
		}
	}

	return
}

func translateOidx(srcOidx int, srcNodesPerOrbit int, destNodesPerOrbit int) int {
	if srcNodesPerOrbit == destNodesPerOrbit {
		return srcOidx
	} else if srcNodesPerOrbit == 12 && destNodesPerOrbit == 16 {
		return []int{0, 1, 3, 4, 5, 7, 8, 9, 11, 12, 13, 15}[srcOidx]
	} else if srcNodesPerOrbit == 16 && destNodesPerOrbit == 12 {
		return []int{0, 1, 1, 2, 3, 4, 4, 5, 6, 7, 7, 8, 9, 10, 10, 11}[srcOidx]
	} else {
		return (srcOidx * destNodesPerOrbit) / srcNodesPerOrbit
	}
}
