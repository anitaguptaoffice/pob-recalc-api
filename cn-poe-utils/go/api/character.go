package api

// Character 角色信息
type Character struct {
	Name          string `json:"name"`
	Realm         string `json:"realm"`
	Class         string `json:"class"`
	League        string `json:"league"`
	Level         int    `json:"level"`
	LastLoginTime int    `json:"lastLoginTime"`
}

// GetCharactersResult 获取角色列表的结果
type GetCharactersResult []Character
