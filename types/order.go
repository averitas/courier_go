package types

type Order struct {
	Id   string `form:"id" json:"id" binding:"required"`
	Name string `form:"name" json:"name" binding:"required"`
	// time in seconds
	PrepTime int `form:"prepTime" json:"prepTime" binding:"required"`
}
