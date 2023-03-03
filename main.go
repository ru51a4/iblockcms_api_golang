package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type iblock struct {
	Id        int      `json:"id" gorm:"primaryKey"`
	Name      string   `json:"name"`
	Slug      string   `json:"slug"`
	Parent_id int      `json:"parent_id"`
	Left      int      `json:"left"`
	Right     int      `json:"right"`
	Depth     int      `json:"depth"`
	Iblock    []iblock `gorm:"foreignkey:Iblock_propertyID"`
}

func (iblock) TableName() string {
	return "iblocks"
}

type iblock_elements struct {
	Id                int                 `json:"id" gorm:"primaryKey"`
	Slug              string              `json:"slug"`
	Name              string              `json:"name"`
	Iblock_prop_value []iblock_prop_value `gorm:"foreignkey:Iblock_elementsID"`
}

func (iblock_elements) TableName() string {
	return "iblock_elements"
}

type iblock_property struct {
	Id                int                 `json:"id" gorm:"primaryKey"`
	Is_number         int                 `json:"is_number"`
	Is_multy          int                 `json:"is_multy"`
	Name              string              `json:"name"`
	IblockId          int                 `gorm:"column:iblock_id;"`
	Iblock_prop_value []iblock_prop_value `gorm:"foreignkey:Iblock_propertyID"`
}

func (iblock_property) TableName() string {
	return "iblock_properties"
}

type iblock_prop_value struct {
	Id                int    `gorm:"primaryKey"`
	Value             string `json:"value"`
	Slug              string `json:"slug"`
	Value_number      int    `json:"value_number"`
	Iblock_propertyID int    `gorm:"column:prop_id;"`
	Iblock_elementsID int    `gorm:"column:el_id;"`
}

func (iblock_prop_value) TableName() string {
	return "iblock_prop_values"
}

func main() {
	dsn := "root:root@tcp(127.0.0.1:3306)/iblockcms?charset=utf8mb4&parseTime=True&loc=Local"
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	app := fiber.New()

	app.Get("/catalog", func(c *fiber.Ctx) error {
		var els []iblock_elements
		db.Limit(5).Preload("Iblock_prop_value").Find(&els)
		return c.JSON(&fiber.Map{
			"success": true,
			"els":     els,
		})
	})

	log.Fatal(app.Listen(":3000"))

}
