package xml

import (
	"fmt"
	"reflect"
	"strings"
)

const enemyShaper = "Pinnacle"

// Config 配置信息
type Config struct {
	// buff
	UseFrenzyCharges             *bool // 狂怒球
	UsePowerCharges              *bool // 暴击球
	UseEnduranceCharges          *bool // 耐力球
	MultiplierGaleForce          *int  // 飓风之力层数
	BuffOnslaught                *bool // 猛攻
	BuffArcaneSurge              *bool // 秘术增强
	BuffUnholyMight              *bool // 不洁之力
	BuffFortification            *bool // 护体
	BuffTailwind                 *bool // 提速尾流
	BuffAdrenaline               *bool // 肾上腺素
	ConditionOnConsecratedGround *bool // 你在奉献地面上？
	// skill
	BrandAttachedToEnemy *bool // 烙印附加在敌人身上？
	ConfigResonanceCount *int  // 三位一体层数
	// enemy de-buff
	ProjectileDistance           *int  // 投射物飞行距离
	ConditionEnemyBlinded        *bool // 敌人被致盲
	OverrideBuffBlinded          *int  // 致盲效果
	ConditionEnemyBurning        *bool // 燃烧
	ConditionEnemyIgnited        *bool // 点燃
	ConditionEnemyChilled        *bool // 敌人被冰缓
	ConditionEnemyChilledEffect  *int  // 冰缓效果
	ConditionEnemyShocked        *bool // 敌人被感电
	ConditionShockEffect         *int  // 感电效果
	ConditionEnemyScorched       *bool // 烧灼
	ConditionScorchedEffect      *int  // 烧灼效果
	ConditionEnemyBrittle        *bool // 易碎
	ConditionBrittleEffect       *int  // 易碎效果
	ConditionEnemySapped         *bool // 筋疲力尽
	ConditionSapEffect           *bool // 筋疲力尽效果
	ConditionEnemyIntimidated    *bool // 恐吓
	ConditionEnemyCrushed        *bool // 碾压
	ConditionEnemyUnnerved       *bool // 恐惧
	ConditionEnemyCoveredInFrost *bool // 冰霜缠身
	ConditionEnemyCoveredInAsh   *bool // 灰烬缠身
	// enemy
	EnemyIsBoss string
}

func NewConfig() *Config {
	return &Config{
		EnemyIsBoss: enemyShaper,
	}
}

// String 返回XML字符串
func (c *Config) String() string {
	var inputs []string
	v := reflect.ValueOf(c).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 跳过未设置的指针字段
		if field.Kind() == reflect.Pointer && !field.IsNil() {
			val := field.Elem().Interface()
			var typeName string
			switch val.(type) {
			case string:
				typeName = "string"
			case int:
				typeName = "number"
			case bool:
				typeName = "boolean"
			default:
				continue
			}
			inputName := strings.ToLower(fieldType.Name[:1]) + fieldType.Name[1:]
			inputs = append(inputs, fmt.Sprintf(`<Input name="%s" %s="%v"/>`, inputName, typeName, val))
		} else if field.Kind() == reflect.String && field.String() != "" {
			inputName := strings.ToLower(fieldType.Name[:1]) + fieldType.Name[1:]
			inputs = append(inputs, fmt.Sprintf(`<Input name="%s" string="%s"/>`, inputName, field.String()))
		}
	}

	inputsView := strings.Join(inputs, "\n")
	return fmt.Sprintf(`<Config>
%s
</Config>`, inputsView)
}
