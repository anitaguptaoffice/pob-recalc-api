package transform

import (
	"fmt"

	"github.com/cn-poe-community/cn-poe-utils/go/api"
	"github.com/cn-poe-community/cn-poe-utils/go/data/pob"
)

// GetSlotName 获取物品信息在POB中的插槽名称
func GetSlotName(itemData *api.Item) (string, error) {
	inventoryId := itemData.InventoryId

	if inventoryId != nil {
		if *inventoryId == "Flask" {
			return fmt.Sprintf("Flask %d", *itemData.X+1), nil
		}

		if slotName, ok := pob.DefaultData.SlotMap[*inventoryId]; ok {
			return slotName, nil
		}
	}

	return "", fmt.Errorf("unknown inventoryId: %v", inventoryId)
}
