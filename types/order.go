package types

type Order struct {
	Id       string `form:"id" json:"id" binding:"required"`
	Name     string `form:"name" json:"name" binding:"required"`
	PrepTime string `form:"prepTime" json:"prepTime" binding:"required"`
}
